package docker

import (
	"context"
	"os/exec"
)

type localProxy struct {
	c *DockerConnect
}

func newLocalProxy(c *DockerConnect) *localProxy {
	return &localProxy{c: c}
}

func (localProxy) connect(ctx context.Context) error {
	return nil
}

func (localProxy) docker(ctx context.Context, args ...string) (cmdAdaptor, error) {
	return exec.CommandContext(ctx, "docker", args...), nil
}
