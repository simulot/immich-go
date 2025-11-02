package e2eutils

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"
)

const timeout = 5 * time.Minute

type ImmichController struct {
	dcPath    string // path to the docker compose directory
	sshHost   string // SSH host for remote server (optional, e.g., "user@hostname:22")
	immichUrl string // immich URL (http://hostname:2283)
}

// Remote configures the controller to use SSH for remote operations
func Remote(sshhost, immichUrl, dcPath string) func(ctx context.Context, ictlr *ImmichController) error {
	return func(ctx context.Context, ictlr *ImmichController) error {
		ictlr.sshHost = sshhost
		ictlr.dcPath = dcPath
		ictlr.immichUrl = immichUrl
		err := ExecWithTimeout(ctx, timeout,
			"ssh", "-o", "StrictHostKeyChecking=no", sshhost, "test", "-f", ictlr.GetDockerComposeFile())
		return err
	}
}

// Local configures the control on the host machine
func Local(p string) func(ctx context.Context, ictlr *ImmichController) error {
	return func(_ context.Context, ictlr *ImmichController) error {
		ictlr.dcPath = p
		ictlr.immichUrl = "http://localhost:2283"
		_, err := os.Stat(ictlr.GetDockerComposeFile())
		return err
	}
}

// IsRemote returns true if the controller is configured for remote SSH access
func (ictlr *ImmichController) IsRemote() bool {
	return ictlr.sshHost != ""
}

// OpenImmichController opens a new ImmichController instance with the specified docker-compose file path
func OpenImmichController(ctx context.Context, initializer func(ctx context.Context, ictlr *ImmichController) error) (*ImmichController, error) {
	ictlr := &ImmichController{}
	err := initializer(ctx, ictlr)
	if err != nil {
		return nil, err
	}
	return ictlr, nil
}

// NewImmichController creates a new controller with the specified path
func NewImmichController(p string) (*ImmichController, error) {
	err := os.MkdirAll(p, 0o755)
	if err != nil {
		return nil, fmt.Errorf("can't make the directory: %w", err)
	}
	return &ImmichController{dcPath: p}, nil
}

func (ictlr *ImmichController) GetDockerComposeFile() string {
	return path.Join(ictlr.dcPath, "docker-compose.yml")
}

// ImmichGet performs a complete Immich setup: downloads configuration files, starts services, and waits for API readiness
func (ictlr *ImmichController) ImmichGet(ctx context.Context) error {
	// Get Immich setup files and prepare Docker environment
	if err := ictlr.DeployImmich(ctx); err != nil {
		return fmt.Errorf("failed to get immich setup: %w", err)
	}

	// Start Immich services
	if err := ictlr.RunImmich(ctx); err != nil {
		return fmt.Errorf("failed to run immich: %w", err)
	}

	// Wait for API to be ready
	if err := ictlr.WaitAPI(ctx, 3*time.Minute); err != nil {
		return fmt.Errorf("failed to wait for immich API: %w", err)
	}

	return nil
}

// dockerCompose executes docker compose commands with the controller's docker-compose file
func (ictlr *ImmichController) dockerCompose(ctx context.Context, args ...string) error {
	// Prepend the docker compose file argument if we have a specific file
	cmdArgs := []string{"compose", "-f", ictlr.GetDockerComposeFile()}
	cmdArgs = append(cmdArgs, args...)
	return ictlr.execCommand(ctx, timeout, "docker", cmdArgs...)
}

// execCommand executes a command either locally or via SSH depending on configuration
func (ictlr *ImmichController) execCommand(ctx context.Context, timeout time.Duration, command string, args ...string) error {
	if !ictlr.IsRemote() {
		return ExecWithTimeout(ctx, timeout, command, args...)
	}

	// Build SSH command
	sshArgs := []string{"-o", "StrictHostKeyChecking=no", ictlr.sshHost}

	// Build the remote command
	remoteCmd := command
	for _, arg := range args {
		// Properly escape arguments for SSH
		if strings.Contains(arg, " ") || strings.Contains(arg, "\n") || strings.Contains(arg, "'") {
			arg = fmt.Sprintf("'%s'", strings.ReplaceAll(arg, "'", "'\\''"))
		}
		remoteCmd += " " + arg
	}
	sshArgs = append(sshArgs, remoteCmd)

	return ExecWithTimeout(ctx, timeout, "ssh", sshArgs...)
}

// DeployImmich downloads Immich configuration files from the application web page
// and prepares the Docker environment
func (ictlr *ImmichController) DeployImmich(ctx context.Context) error {
	if ictlr.IsRemote() {
		return errors.New("can't deploy on a remote host")
	}
	err := os.MkdirAll(ictlr.dcPath, 0o755)
	if err != nil {
		return err
	}

	// purge any previous instance
	_, err = os.Stat(ictlr.GetDockerComposeFile())
	if err == nil {
		err = ictlr.StopImmich(ctx)
		if err == nil {
			err = ExecWithTimeout(ctx, timeout, "docker", "system", "prune", "-f")
		}
		if err != nil {
			return fmt.Errorf("can't get immich: %w", err)
		}
	}

	ef, err := GetFile(ctx, "https://github.com/immich-app/immich/releases/latest/download/example.env")
	if err != nil {
		return fmt.Errorf("can't get immich: %w", err)
	}
	err = os.WriteFile(path.Join(ictlr.dcPath, ".env"), ef, 0o755)
	if err != nil {
		return fmt.Errorf("can't get immich: %w", err)
	}

	df, err := GetAndTransformDockerFile(ctx, "https://github.com/immich-app/immich/releases/latest/download/docker-compose.yml")
	if err != nil {
		return fmt.Errorf("can't get immich: %w", err)
	}
	err = os.WriteFile(ictlr.GetDockerComposeFile(), df, 0o755)
	if err != nil {
		return fmt.Errorf("can't get immich: %w", err)
	}

	err = ictlr.dockerCompose(ctx, "pull", "-q")
	if err != nil {
		return fmt.Errorf("can't get immich: %w", err)
	}

	err = ictlr.RunImmich(ctx)
	if err != nil {
		return fmt.Errorf("can't get immich: %w", err)
	}

	err = ictlr.WaitAPI(ctx, 3*time.Minute)
	return err
}

// GetFile downloads a file from the given URL with context support for cancellation
func GetFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("can't get file at %q: %s", url, resp.Status)
	}
	defer resp.Body.Close()

	if resp.Body != nil {
		slog.Info("http.get", "url", url, "status", resp.Status)
		return io.ReadAll(resp.Body)
	}
	return nil, errors.New("empty content")
}

func GetAndTransformDockerFile(ctx context.Context, url string) ([]byte, error) {
	df, err := GetFile(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("can't get docker file: %w", err)
	}

	re := regexp.MustCompile(`\$\{[^}]*LOCATION\}`)
	out := bytes.NewBuffer(nil)

	for l := range strings.Lines(string(df)) {
		if strings.Contains(l, "_LOCATION}") {
			_, _ = io.WriteString(out, "      # immich-go e2eTests: keep everything inside the container\n")
			l = re.ReplaceAllString(l, "local-volume")
		}
		_, _ = io.WriteString(out, l)
	}
	_, _ = io.WriteString(out, "  local-volume:\n")
	return out.Bytes(), nil
}

// RunImmich starts the Immich services using docker compose
func (ictlr *ImmichController) RunImmich(ctx context.Context) error {
	err := ictlr.dockerCompose(ctx, "up", "-d", "--build", "--renew-anon-volumes", "--force-recreate", "--remove-orphans")
	if err != nil {
		return fmt.Errorf("can't run immich: %w", err)
	}
	return nil
}

// StopImmich stops the Immich services using docker compose
func (ictlr *ImmichController) StopImmich(ctx context.Context) error {
	err := ictlr.dockerCompose(ctx, "down", "--volumes", "--remove-orphans")
	if err != nil {
		return fmt.Errorf("can't stop immich: %w", err)
	}
	return nil
}

// PauseImmichServer stops the Immich server container specifically
func (ictlr *ImmichController) PauseImmichServer(ctx context.Context) error {
	err := ictlr.dockerCompose(ctx, "stop", "immich-server")
	if err != nil {
		return fmt.Errorf("can't stop immich-server: %w", err)
	}
	return nil
}

// ResumeImmichServer starts the Immich server container in detached mode
func (ictlr *ImmichController) ResumeImmichServer(ctx context.Context) error {
	err := ictlr.dockerCompose(ctx, "up", "-d")
	if err != nil {
		return fmt.Errorf("can't spin up immich: %w", err)
	}
	return ictlr.WaitAPI(ctx, 30*time.Second)
}

// WaitAPI waits for the Immich API to become available by polling the ping endpoint during 30 seconds
func (ictlr *ImmichController) WaitAPI(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err == context.DeadlineExceeded {
				slog.Error("immich API is not ready")
				return err
			}
			return ctx.Err()
		default:
			slog.Info("pinging the immich API...")
			err := ictlr.PingAPI(ctx)
			if err == nil {
				slog.Info("immich API is ready")
				return nil
			}
		}
		time.Sleep(5 * time.Second)
	}
}

// PingAPI performs a quick health check on the Immich API server
func (ictlr *ImmichController) PingAPI(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ictlr.immichUrl+"/api/server/ping", nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// ExecWithTimeout executes a command with a timeout and context support
func ExecWithTimeout(ctx context.Context, timeout time.Duration, command string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	slog.Info("exec", "command", strings.Join(append([]string{command}, args...), " "))
	cmd := exec.CommandContext(ctx, command, args...)

	rc, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	logOuput(ctx, rc)

	rc, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}
	logOuput(ctx, rc)

	err = cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	return err
}

func logOuput(ctx context.Context, r io.Reader) {
	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			level := slog.LevelInfo
			if strings.Contains(s.Text(), "error") {
				level = slog.LevelError
			}
			slog.Log(ctx, level, s.Text())
		}
	}()
}

// RunWithTimeout runs a function with a timeout context
func RunWithTimeout(timeout time.Duration, f func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return f(ctx)
	}
}
