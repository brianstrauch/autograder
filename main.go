package main

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	if err := pullImages(); err != nil {
		panic(err)
	}

	log.Fatal(Run())
}

// Pull an image for each supported language.
func pullImages() error {
	ctx := context.Background()

	docker, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	images := []string{
		"docker.io/library/python",
	}

	for _, image := range images {
		if _, err := docker.ImagePull(ctx, image, types.ImagePullOptions{}); err != nil {
			return err
		}
	}

	return nil
}
