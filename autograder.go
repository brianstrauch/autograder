package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/brianstrauch/autograder/errors"
	"github.com/docker/docker/client"
)

type Autograder struct {
	Jobs        []*Job
	runningJobs map[int]*Job
}

func Run() error {
	a := &Autograder{
		runningJobs: make(map[int]*Job),
	}

	go a.manageJobs()

	http.Handle("/upload", errors.ErrorHandler(a.UploadProgram))
	http.Handle("/job", errors.ErrorHandler(a.GetJob))

	const port = 1024
	addr := fmt.Sprintf(":%d", port)
	log.Println("Autograder listening at http://localhost" + addr)
	return http.ListenAndServe(addr, nil)
}

// Upload a program file, create a job, and queue.
func (a *Autograder) UploadProgram(w http.ResponseWriter, r *http.Request) *errors.Error {
	if err := r.ParseForm(); err != nil {
		return errors.NewInternalError(err)
	}

	file, header, err := r.FormFile("program")
	if err != nil {
		return &errors.Error{
			Code:    http.StatusBadRequest,
			Message: "Please upload a program.",
			Err:     err,
		}
	}

	const fileSizeLimit = 1024
	if header.Size > fileSizeLimit {
		return &errors.Error{
			Code:    http.StatusBadRequest,
			Message: "File is larger than 1MB.",
		}
	}

	id := len(a.Jobs)

	job := NewJob(id, file)
	a.Jobs = append(a.Jobs, job)
	a.runningJobs[id] = job

	if err := json.NewEncoder(w).Encode(job); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

// Get a running job from its ID.
func (a *Autograder) GetJob(w http.ResponseWriter, r *http.Request) *errors.Error {
	val := r.URL.Query().Get("id")
	if val == "" {
		return &errors.Error{
			Code:    http.StatusBadRequest,
			Message: "No ID provided.",
		}
	}

	id, err := strconv.Atoi(val)
	if err != nil {
		return &errors.Error{
			Code:    http.StatusBadRequest,
			Message: "ID must be an integer.",
			Err:     err,
		}
	}

	if id < 0 || id >= len(a.Jobs) {
		return &errors.Error{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Job %d does not exist.", id),
		}
	}

	if err := json.NewEncoder(w).Encode(a.Jobs[id]); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

// Keep track of running jobs, refresh once per second.
func (a *Autograder) manageJobs() {
	docker, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	for _ = range time.NewTicker(time.Second).C {
		for id, job := range a.runningJobs {
			switch job.State {
			case "READY":
				job.State = "RUNNING"
				go job.run(docker)
			case "ALIVE":
				continue
			case "RIGHT", "WRONG", "ERROR":
				delete(a.runningJobs, id)
			}
		}
	}
}
