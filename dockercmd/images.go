package dockercmd

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
)

func (dc *DockerClient) ListImages() []image.Summary {
	images, err := dc.cli.ImageList(context.Background(), image.ListOptions{ContainerCount: true})

	if err != nil {
		panic(err)
	}

	return images
}

func (dc *DockerClient) DeleteImage(id string, opts image.RemoveOptions) error {
	_, err := dc.cli.ImageRemove(context.Background(), id, opts)
	return err
}

func (dc *DockerClient) PruneImages() (types.ImagesPruneReport, error) {
	report, err := dc.cli.ImagesPrune(context.Background(), filters.Args{})
	return report, err
}
