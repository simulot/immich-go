package tester

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Engines supported by the e2e tooling. Both accept the same command shape:
// "<engine> compose ..." and "<engine> volume ...".
const (
	EngineDocker = "docker"
	EnginePodman = "podman"
)

// Engine returns the container engine binary to run.
// E2E_ENGINE selects it, with the same values as the e2e shell scripts.
// Docker is the default, so an unset variable keeps the previous behaviour.
func Engine() string {
	switch engine := os.Getenv("E2E_ENGINE"); engine {
	case EngineDocker, EnginePodman:
		return engine
	default:
		return EngineDocker
	}
}

// EngineRun runs one command of the container engine.
func (s *ImmichController) EngineRun(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, Engine(), args...).CombinedOutput()
}

// Compose runs the Compose v2 front end of the engine on the test compose file.
func (s *ImmichController) Compose(ctx context.Context, args ...string) error {
	o, err := s.EngineRun(ctx, append([]string{"compose", "--file=" + s.ComposePath, "--env-file=" + s.ComposeEnvPath}, args...)...)
	if err != nil {
		return fmt.Errorf("compose error:\n%s\n%w", string(o), err)
	}
	return nil
}

// Volumes lists the volume names held by the engine.
func (s *ImmichController) Volumes(ctx context.Context) ([]string, error) {
	out, err := s.EngineRun(ctx, "volume", "ls", `--format={{.Name}}`)
	if err != nil {
		return nil, fmt.Errorf("can't get the volume list:\n%s\n%w", string(out), err)
	}
	list := strings.TrimSpace(string(out))
	if list == "" {
		return nil, nil
	}
	return strings.Split(list, "\n"), nil
}

// SudoRun runs a command as root through sudo.
func SudoRun(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.Command("sudo", args...)
	return cmd.CombinedOutput()
}
