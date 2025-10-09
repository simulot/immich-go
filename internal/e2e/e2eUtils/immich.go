package e2eutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"
)

const timeout = 1 * time.Minute

type ImmichController struct {
	dcFile string // path to the docker compose file
}

func NewImmichController(p string) (*ImmichController, error) {
	s, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("can't get file info: %w", err)
	}
	if s.IsDir() {
		p = path.Join(p, "docker-compose.yml")
		s, err = os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("can't get file info: %w", err)
		}

	}
	return &ImmichController{dcFile: p}, nil
}

func (ictlr *ImmichController) ImmichGet(ctx context.Context) error {
	// Get Immich setup files and prepare Docker environment
	if err := ictlr.GetImmich(ctx); err != nil {
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

func (ictlr *ImmichController) dockerCompose(ctx context.Context, args ...string) ([]byte, error) {
	// Prepend the docker compose file argument if we have a specific file
	cmdArgs := []string{"compose"}
	if ictlr.dcFile != "" {
		cmdArgs = append(cmdArgs, "-f", ictlr.dcFile)
	}
	cmdArgs = append(cmdArgs, args...)
	return ExecWithTimeout(ctx, timeout, "docker", cmdArgs...)
}

func (ictlr *ImmichController) GetImmich(ctx context.Context) error {
	err := os.MkdirAll(ictlr.dcFile, 0o755)
	if err != nil {
		return err
	}

	err = os.Chdir(ictlr.dcFile)
	if err != nil {
		return err
	}

	ef, err := GetFile(ctx, "https://github.com/immich-app/immich/releases/latest/download/example.env")
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(ictlr.dcFile, ".env"), ef, 0o755)
	if err != nil {
		return err
	}

	df, err := GetFile(ctx, "https://github.com/immich-app/immich/releases/latest/download/docker-compose.yml")
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(ictlr.dcFile, "docker-compose.yml"), df, 0o755)
	if err != nil {
		return err
	}

	out, err := ExecWithTimeout(ctx, timeout, "docker", "system", "prune", "-f")
	if err != nil {
		return fmt.Errorf("docker: %s", out)
	}

	out, err = ictlr.dockerCompose(ctx, "pull")
	if err != nil {
		return fmt.Errorf("docker: %s", out)
	}

	return nil
}

func GetFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.Body != nil {
		return io.ReadAll(resp.Body)
	}
	return nil, errors.New("empty content")
}

func (ictlr *ImmichController) RunImmich(ctx context.Context) error {
	out, err := ictlr.dockerCompose(ctx, "up", "--build", "--renew-anon-volumes", "--force-recreate", "--remove-orphans")
	if err != nil {
		return fmt.Errorf("docker: %s", out)
	}
	return nil
}

func (ictlr *ImmichController) StopImmich(ctx context.Context) error {
	out, err := ictlr.dockerCompose(ctx, "stop")
	if err != nil {
		return fmt.Errorf("docker: %s", out)
	}
	return nil
}

func (ictlr *ImmichController) PauseImmichServer(ctx context.Context) error {
	out, err := ictlr.dockerCompose(ctx, "stop", "immich-server")
	if err != nil {
		return fmt.Errorf("docker: %s", out)
	}
	return nil
}

func (ictlr *ImmichController) ResumeImmichServer(ctx context.Context) error {
	out, err := ictlr.dockerCompose(ctx, "up", "-d")
	if err != nil {
		return fmt.Errorf("docker: %s", out)
	}
	return nil
}

func (ictlr *ImmichController) WaitAPI(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	for {
		err := ictlr.PingAPI(ctx)
		if err == context.DeadlineExceeded {
			return err
		}
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func (ictlr *ImmichController) PingAPI(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:2283/api/server/ping", nil)
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

// func (ic *ImmichController) COMMANDImmich(ctx context.Context) error {
// 	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", "immich_postgres", "psql", "--dbname=immich", "--username=postgres", "-c", "select 1")
// 	out, err := ExecWithTimeout(ctx, cmd, timeout)
// 	if err != nil {
// 		return fmt.Errorf("docker: %s", out)
// 	}
// 	return nil
// }

func ExecWithTimeout(ctx context.Context, timeout time.Duration, command string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	slog.Info("exec", "command", command, "args", args)
	cmd := exec.CommandContext(ctx, command, args...)

	// Run the command and capture output
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("exec error", "output", out)
	}
	return out, err
}

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
