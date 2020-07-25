package main

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var languages = map[string]language{
	"python": {
		image:    "docker.io/library/python",
		filename: "main.py",
		command:  []string{"python", "main.py", "<", inputFile},
	},
}

type language struct {
	image    string
	filename string
	command  []string
}

// Pull an image for each supported language.
func pullImages() error {
	ctx := context.Background()

	docker, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	for _, language := range languages {
		if _, err := docker.ImagePull(ctx, language.image, types.ImagePullOptions{}); err != nil {
			return err
		}
	}

	return nil
}
