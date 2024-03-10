package tester

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// run the docker command
func (s *ImmichController) DockerRun(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
}

// run the docker compose command
func (s *ImmichController) DockerCompose(ctx context.Context, args ...string) error {
	o, err := s.DockerRun(ctx, append([]string{"compose", "--file=" + s.ComposePath, "--env-file=" + s.ComposeEnvPath}, args...)...)
	if err != nil {
		return fmt.Errorf("DockerCompose error:\n%s\n%w", string(o), err)
	}
	return err
}

// run the docker volume ls command
func (s *ImmichController) DockerVolumes(ctx context.Context) ([]string, error) {
	out, err := s.DockerRun(ctx, "volume", "ls", `--format={{.Name}}`)
	if err != nil {
		return nil, fmt.Errorf("can't get docker volumes:\n%s\n%w", string(out), err)
	}
	volumes := strings.Split(string(out), "\n")
	return volumes, err
}

func SudoRun(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.Command("sudo", args...)
	return cmd.CombinedOutput()
}
