package docker

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os/user"
	"path/filepath"

	"github.com/melbahja/goph"
)

type sshProxy struct {
	sshAuth   goph.Auth
	sshUser   string
	sshHost   string
	sshClient *goph.Client
}

func newSSHProxy(host string) (*sshProxy, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ssh" {
		return nil, fmt.Errorf("unsupported protocol %s: %s", u.Scheme, host)
	}
	p := sshProxy{}
	p.sshUser = u.User.Username()
	p.sshHost = u.Host
	pass, set := u.User.Password()
	if set {
		p.sshAuth = goph.Password(pass)
	} else {
		var u *user.User
		u, err = user.Current()
		if err != nil {
			return nil, err
		}
		if p.sshUser == "" {
			p.sshUser = u.Username
		}
		keyFile := filepath.Join(u.HomeDir, ".ssh", "id_rsa")
		p.sshAuth, err = goph.Key(keyFile, "")
		if err != nil {
			return nil, err
		}
	}
	return &p, nil
}

func (p *sshProxy) connect(ctx context.Context) error {
	var err error
	p.sshClient, err = goph.New(p.sshUser, p.sshHost, p.sshAuth)
	return err
}

func (p *sshProxy) docker(ctx context.Context, args ...string) (cmdAdaptor, error) {
	cmd, err := p.sshClient.CommandContext(ctx, "docker", args...)
	return &sshCmd{Cmd: cmd}, err
}

// sshCmd shim
type sshCmd struct {
	*goph.Cmd
}

func (c *sshCmd) StdoutPipe() (io.ReadCloser, error) {
	r, err := c.Cmd.StdoutPipe()
	return io.NopCloser(r), err
}
