package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/docker/client"

	"github.com/docker/docker/api/types"
)

type Server struct {
	jobs        []*Job
	runningJobs map[int]*Job
}

func main() {
	if err := pullImages(); err != nil {
		panic(err)
	}

	s := &Server{runningJobs: make(map[int]*Job)}
	go s.manageJobs()

	http.Handle("/upload", ErrorHandler(s.uploadProgram))
	http.Handle("/job", ErrorHandler(s.getJob))

	const port = 1024
	addr := fmt.Sprintf(":%d", port)
	log.Println("Server listening at http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
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

// Upload a program file, create a job, and queue.
// POST /upload { program } -> { job }
func (s *Server) uploadProgram(w http.ResponseWriter, r *http.Request) *Error {
	if err := r.ParseForm(); err != nil {
		return NewInternalError(err)
	}

	file, header, err := r.FormFile("program")
	if err != nil {
		return &Error{
			Code:    http.StatusBadRequest,
			Message: "Please upload a program.",
			Err:     err,
		}
	}

	const fileSizeLimit = 1024
	if header.Size > fileSizeLimit {
		return &Error{
			Code:    http.StatusBadRequest,
			Message: "File is larger than 1MB.",
		}
	}

	id := len(s.jobs)

	job := NewJob(id, file)
	s.jobs = append(s.jobs, job)
	s.runningJobs[id] = job

	if err := json.NewEncoder(w).Encode(job); err != nil {
		return NewInternalError(err)
	}

	return nil
}

// Get a running job from its ID.
// GET /job?id=0 -> { job }
func (s *Server) getJob(w http.ResponseWriter, r *http.Request) *Error {
	val := r.URL.Query().Get("id")
	if val == "" {
		return &Error{
			Code:    http.StatusBadRequest,
			Message: "No ID provided.",
		}
	}

	id, err := strconv.Atoi(val)
	if err != nil {
		return &Error{
			Code:    http.StatusBadRequest,
			Message: "ID must be an integer.",
			Err:     err,
		}
	}

	if id < 0 || id >= len(s.jobs) {
		return &Error{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Job %d does not exist.", id),
		}
	}

	if err := json.NewEncoder(w).Encode(s.jobs[id]); err != nil {
		return NewInternalError(err)
	}

	return nil
}

// Keep track of running jobs, refresh once per second.
func (s *Server) manageJobs() {
	docker, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	for _ = range time.NewTicker(time.Second).C {
		for id, job := range s.runningJobs {
			log.Println(id, job.State)

			switch job.State {
			case "READY":
				job.State = "RUNNING"
				go job.run(docker)
			case "ALIVE":
				continue
			case "RIGHT", "WRONG", "ERROR":
				delete(s.runningJobs, id)
			}
		}
	}
}
