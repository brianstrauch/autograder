package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type language struct {
	image    string
	filename string
	command  []string
}

var languageInfo = map[string]language{
	"sed": {
		image:    "docker.io/library/alpine",
		filename: "script",
		command:  []string{"sed", "-f", "script", inputFile},
	},
	"python": {
		image:    "docker.io/library/python",
		filename: "main.py",
		command:  []string{"python", "main.py", "<", inputFile},
	},
}

func main() {
	if err := pullImages(); err != nil {
		panic(err)
	}

	a := NewAutograder()

	go a.ManageJobs()

	routes := map[string]func(w http.ResponseWriter, r *http.Request) *APIError{
		"/file": a.UploadProgramFile,
		"/text": a.UploadProgramText,
		"/job":  a.GetJob,
	}

	for pattern, handler := range routes {
		http.Handle(pattern, CustomHandler(handler))
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

type CustomHandler func(w http.ResponseWriter, r *http.Request) *APIError

func (h CustomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)

	if err := h(w, r); err != nil {
		log.Println(err.Code, err.Message)
		if err.Err != nil {
			log.Println(err.Err)
		}

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

	for _, info := range languageInfo {
		if _, err := docker.ImagePull(ctx, info.image, types.ImagePullOptions{}); err != nil {
			return err
		}
	}

	return nil
}
