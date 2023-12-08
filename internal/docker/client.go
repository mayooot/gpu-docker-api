package docker

import (
	"github.com/docker/docker/client"
)

var (
	defaultHost = "unix:///var/run/docker.sock"
	Cli         *client.Client
)

func InitDockerClient() (err error) {
	Cli, err = client.NewClientWithOpts(client.WithHost(defaultHost), client.WithAPIVersionNegotiation())
	return err
}

func CloseDockerClient() {
	Cli.Close()
}
