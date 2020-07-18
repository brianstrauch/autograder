package main

import (
	"context"
	"fmt"
	"github.com/brianstrauch/autograder/errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	if err := pullImages(); err != nil {
		panic(err)
	}

	a := NewAutograder()

	go a.ManageJobs()

	routes := map[string]func(w http.ResponseWriter, r *http.Request) *errors.APIError{
		"/file": a.UploadProgramFile,
		"/text": a.UploadProgramText,
		"/job":  a.GetJob,
	}

	for pattern, function := range routes {
		http.Handle(pattern, errors.ErrorHandler(function))
	}

	port := 1024
	if str := os.Getenv("PORT"); str != "" {
		var err error
		port, err = strconv.Atoi(str)
		if err != nil {
			panic(err)
		}
	}

	addr := fmt.Sprintf(":%d", port)
	log.Println("Listening at http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// Pull an image for each supported language.
func pullImages() error {
	ctx := context.Background()

	docker, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	for _, info := range languageInfo {
		if _, err := docker.ImagePull(ctx, info.image, types.ImagePullOptions{}); err != nil {
			return err
		}
	}

	return nil
}
