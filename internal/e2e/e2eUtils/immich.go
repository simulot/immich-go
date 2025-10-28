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

const timeout = 1 * time.Minute

type ImmichController struct {
	dcPath     string // path to the docker compose directory
	dcFile     string // path to the docker compose file
	sshHost    string // SSH host for remote server (optional, e.g., "user@hostname")
	sshPort    string // SSH port (optional, defaults to 22)
	sshKeyPath string // SSH key path (optional)
}

// WithSSH configures the controller to use SSH for remote operations
func (ictlr *ImmichController) WithSSH(host, port, keyPath string) *ImmichController {
	ictlr.sshHost = host
	ictlr.sshPort = port
	if ictlr.sshPort == "" {
		ictlr.sshPort = "22"
	}
	ictlr.sshKeyPath = keyPath
	return ictlr
}

// IsRemote returns true if the controller is configured for remote SSH access
func (ictlr *ImmichController) IsRemote() bool {
	return ictlr.sshHost != ""
}

// OpenImmichController opens a new ImmichController instance with the specified docker-compose file path
func OpenImmichController(p string) (*ImmichController, error) {
	s, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("can't get file info: %w", err)
	}
	if s.IsDir() {
		p = path.Join(p, "docker-compose.yml")
		_, err = os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("can't get file info: %w", err)
		}
	}
	d, f := path.Split(p)
	return &ImmichController{dcFile: f, dcPath: d}, nil
}

// NewImmichController creates a new controller with the specified path
func NewImmichController(p string) (*ImmichController, error) {
	err := os.MkdirAll(p, 0o755)
	if err != nil {
		return nil, fmt.Errorf("can't make the directory: %w", err)
	}
	return &ImmichController{dcFile: "docker-compose.yml", dcPath: p}, nil
}

func (ictlr *ImmichController) GetDockerComposeFile() string {
	return path.Join(ictlr.dcPath, ictlr.dcFile)
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
	if err := ictlr.WaitAPI(ctx); err != nil {
		return fmt.Errorf("failed to wait for immich API: %w", err)
	}

	return nil
}

// dockerCompose executes docker compose commands with the controller's docker-compose file
func (ictlr *ImmichController) dockerCompose(ctx context.Context, args ...string) error {
	// Prepend the docker compose file argument if we have a specific file
	cmdArgs := []string{"compose"}
	if ictlr.dcFile != "" {
		cmdArgs = append(cmdArgs, "-f", ictlr.GetDockerComposeFile())
	}
	cmdArgs = append(cmdArgs, args...)
	return ictlr.execCommand(ctx, timeout, "docker", cmdArgs...)
}

// execCommand executes a command either locally or via SSH depending on configuration
func (ictlr *ImmichController) execCommand(ctx context.Context, timeout time.Duration, command string, args ...string) error {
	if !ictlr.IsRemote() {
		return ExecWithTimeout(ctx, timeout, command, args...)
	}

	// Build SSH command
	sshArgs := []string{}
	if ictlr.sshPort != "" && ictlr.sshPort != "22" {
		sshArgs = append(sshArgs, "-p", ictlr.sshPort)
	}
	if ictlr.sshKeyPath != "" {
		sshArgs = append(sshArgs, "-i", ictlr.sshKeyPath)
	}
	sshArgs = append(sshArgs, ictlr.sshHost)

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

// DeployImmich downloads Immich configuration files and prepares the Docker environment
func (ictlr *ImmichController) DeployImmich(ctx context.Context) error {
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
	err = os.WriteFile(path.Join(ictlr.dcPath, ictlr.dcFile), df, 0o755)
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

	err = ictlr.WaitAPI(ctx)
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
			_, _ = io.WriteString(out, "      # immicch-go e2eTests: keep everything inside the container\n")
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
	return nil
}

// WaitAPI waits for the Immich API to become available by polling the ping endpoint
func (ictlr *ImmichController) WaitAPI(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	for {
		slog.Info("pinging the immich API...")
		err := ictlr.PingAPI(ctx)
		if err == context.DeadlineExceeded {
			slog.Error("immich API is not ready")
			return err
		}
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	slog.Info("immich API is ready")
	return nil
}

// PingAPI performs a quick health check on the Immich API server
func (ictlr *ImmichController) PingAPI(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// Get the Immich URL from environment variable (set by CI workflow)
	pingURL := os.Getenv("E2E_IMMICH_URL")
	if pingURL != "" {
		// Use the full URL from environment (e.g., http://100.x.x.x:2283)
		pingURL = pingURL + "/api/server/ping"
	} else {
		// Fallback to localhost for local development
		host := "localhost"
		port := "2283"
		pingURL = fmt.Sprintf("http://%s:%s/api/server/ping", host, port)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingURL, nil)
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
