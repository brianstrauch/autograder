package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/brianstrauch/autograder/internal/errors"
	"github.com/brianstrauch/autograder/pkg"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	if err := pullImages(); err != nil {
		panic(err)
	}

	a := pkg.NewAutograder()

	go a.ManageJobs()

	routes := map[string]func(w http.ResponseWriter, r *http.Request) *errors.Error{
		"/file": a.UploadProgramFile,
		"/text": a.UploadProgramText,
		"/job":  a.GetJob,
	}

	for pattern, handler := range routes {
		http.Handle(pattern, CustomHandler(handler))
	}

	const port = 1024
	addr := fmt.Sprintf(":%d", port)
	log.Println("Listening at http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

type CustomHandler func(w http.ResponseWriter, r *http.Request) *errors.Error

func (h CustomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)

	if err := h(w, r); err != nil {
		log.Println(err.Code, err.Message, err.Err)

		w.WriteHeader(err.Code)
		err.Err = nil
		if err := json.NewEncoder(w).Encode(err); err != nil {
			log.Println(err)
		}
	}
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
