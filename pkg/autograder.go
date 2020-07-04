package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/brianstrauch/autograder/internal/errors"
	"github.com/docker/docker/client"
)

type Autograder struct {
	Jobs        []*Job
	runningJobs map[int]*Job
}

type Upload struct {
	Problem  string `json:"problem"`
	Language string `json:"language"`
	Text     string `json:"text"`
}

func NewAutograder() *Autograder {
	return &Autograder{
		runningJobs: make(map[int]*Job),
	}
}

// Upload a program file, create a job, and add to queue.
func (a *Autograder) UploadProgramFile(w http.ResponseWriter, r *http.Request) *errors.Error {
	if err := r.ParseForm(); err != nil {
		return errors.NewInternalError(err)
	}

	problem := r.FormValue("problem")
	if problem == "" {
		return &errors.Error{
			Code:    http.StatusBadRequest,
			Message: "No problem provided.",
		}
	}

	language := r.FormValue("language")
	if language == "" {
		return &errors.Error{
			Code:    http.StatusBadRequest,
			Message: "No language provided.",
		}
	}

	file, header, err := r.FormFile("file")
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

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.NewInternalError(err)
	}

	upload := &Upload{
		Problem:  problem,
		Language: language,
		Text:     string(data),
	}

	job := a.queueJob(upload)

	if err := json.NewEncoder(w).Encode(job); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

func (a *Autograder) UploadProgramText(w http.ResponseWriter, r *http.Request) *errors.Error {
	upload := new(Upload)
	err := json.NewDecoder(r.Body).Decode(&upload)
	if err != nil {
		fmt.Println("Here")
		return errors.NewInternalError(err)
	}

	job := a.queueJob(upload)

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
func (a *Autograder) ManageJobs() {
	docker, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	for _ = range time.NewTicker(time.Second).C {
		for id, job := range a.runningJobs {
			switch job.Status {
			case "READY":
				job.Status = "RUNNING"
				go job.run(docker)
			case "ALIVE":
				continue
			case "RIGHT", "WRONG", "ERROR":
				delete(a.runningJobs, id)
			}
		}
	}
}

func (a *Autograder) queueJob(upload *Upload) *Job {
	id := len(a.Jobs)

	job := NewJob(id, upload)
	a.Jobs = append(a.Jobs, job)
	a.runningJobs[id] = job

	return job
}
