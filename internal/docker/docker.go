package docker

/*
	This package implements a connection to a local docker demon running Immich or a distant one through a SSH connection
	It can:
	- run any docker command
	- download a file from a container
	- upload a fil in a container
*/

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"
)

// DockerConnect represent a connection to a docker service
type DockerConnect struct {
	Host      string
	Container string
	proxy     dockerProxy
}

// dockerProxy in the an interface for a docker command runner
type dockerProxy interface {
	connect(context.Context) error
	docker(context.Context, ...string) (cmdAdaptor, error)
}

// cmdAdaptor provide a common interface between exec.Cmd and goph.Cmd
type cmdAdaptor interface {
	// CommandContext( context.Context,string,...string)
	Run() error
	Start() error
	StdoutPipe() (io.ReadCloser, error)
	StdinPipe() (io.WriteCloser, error)
	CombinedOutput() ([]byte, error)
	Wait() error
}

// NewDockerConnection create a connection with a docker service based on host and container parameters
// host:
// 	- Leave host empty for a local docker
//  - ssh url  for a remote docker on a machine reachable via ssh
//    - ssh://host to access with your username, and you private key
//    - ssh://root@host to access with root user, and you private key
//    - ssh://user:password@host to access with the user and the password

func NewDockerConnection(ctx context.Context, host string, container string) (*DockerConnect, error) {
	d := DockerConnect{
		Host:      host,
		Container: container,
	}

	err := d.connect(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("can't open docker: %w", err)
	}
	return &d, nil
}

// Connect test the connection with docker, and get instance parameters
func (d *DockerConnect) connect(ctx context.Context, host string) error {
	var err error
	if host == "" || host == "local" {
		d.proxy = newLocalProxy(d)
	} else {
		d.proxy, err = newSSHProxy(host)
	}
	if err != nil {
		return err
	}
	err = d.proxy.connect(ctx)
	if err != nil {
		return err
	}

	cmd, err := d.proxy.docker(ctx, "ps", "--format", "{{.Names}}")
	if err != nil {
		return fmt.Errorf("can't connect to the local docker: %s", err)
	}

	b, err := cmd.CombinedOutput()
	buf := bytes.NewBuffer(b)
	for {
		l, err := buf.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		if l[:len(l)-1] == d.Container {
			return nil
		}
	}
	return fmt.Errorf("container 'immich_server' not found: %w", err)
}

// Download a file from the docker container
func (d *DockerConnect) Download(ctx context.Context, hostFile string) (io.Reader, error) {
	cmd, err := d.proxy.docker(ctx, "cp", d.Container+":"+hostFile, "-")
	if err != nil {
		return nil, err
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		defer func() {
			_ = cmd.Wait()
			pw.Close()
		}()
		tr := tar.NewReader(out)
		for {
			hd, err := tr.Next()
			if err == io.EOF {
				break // End of archive
			}
			_ = hd
			if err != nil {
				return
			}
			if _, err := io.Copy(pw, tr); err != nil {
				return
			}
		}
	}()

	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// Upload a file to the docker container
func (d *DockerConnect) Upload(ctx context.Context, file string, size int64, r io.Reader) error {
	cmd, err := d.proxy.docker(ctx, "cp", "-", d.Container+":"+path.Dir(file))
	if err != nil {
		return err
	}

	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		var err error
		tw := tar.NewWriter(in)
		defer func() {
			tw.Close()
			in.Close()
			wg.Done()
		}()
		hdr := tar.Header{
			Name:    path.Base(file),
			Mode:    0o644,
			Size:    size,
			ModTime: time.Now(),
		}
		err = tw.WriteHeader(&hdr)
		if err != nil {
			return
		}
		_, err = io.Copy(tw, r)
		if err != nil {
			return
		}
	}()

	err = cmd.Start()
	if err != nil {
		return err
	}
	wg.Wait()
	err = cmd.Wait()

	return err
}

func (d *DockerConnect) BatchUpload(ctx context.Context, dir string) (*batchUploader, error) {
	f := bytes.NewBuffer(nil)

	cmd, err := d.proxy.docker(ctx, "cp", "-", d.Container+":"+dir)
	if err != nil {
		return nil, err
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	go func() {
		_, _ = io.Copy(os.Stdout, out)
	}()
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	mw := io.MultiWriter(f, in)

	loader := batchUploader{
		fileChannel: make(chan file),
		fileErr:     make(chan error),
	}

	go func() {
		var err error
		tw := tar.NewWriter(mw)
		defer func() {
			// f.Close()
			tw.Close()
			in.Close()
			_ = cmd.Wait()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case f, isOpen := <-loader.fileChannel:
				if !isOpen {
					return
				}

				hdr := tar.Header{
					Name:    f.name,
					Mode:    0o644,
					Size:    int64(len(f.content)),
					ModTime: time.Now(),
				}
				err = tw.WriteHeader(&hdr)
				if err != nil {
					loader.fileErr <- err
					return
				}

				_, err = tw.Write(f.content)
				if err != nil {
					loader.fileErr <- err
					return
				}
				loader.fileErr <- nil
			}
		}
	}()

	err = cmd.Start()

	return &loader, err
}

type batchUploader struct {
	fileChannel chan file
	fileErr     chan error
}

func (b *batchUploader) Upload(name string, content []byte) error {
	b.fileChannel <- file{
		name:    name,
		content: content,
	}
	err := <-b.fileErr
	return err
}

func (b *batchUploader) Close() error {
	close(b.fileChannel)
	return nil
}

type file struct {
	name    string
	content []byte
}

func (d *DockerConnect) Command(ctx context.Context, args ...string) (string, error) {
	cmd, err := d.proxy.docker(ctx, args...)
	if err != nil {
		return "", err
	}

	buffOut := bytes.NewBuffer(nil)

	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	go func() {
		_, _ = io.Copy(buffOut, out)
	}()
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	return buffOut.String(), nil
}
