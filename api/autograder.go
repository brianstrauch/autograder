package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

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
func (a *Autograder) UploadProgramFile(w http.ResponseWriter, r *http.Request) *APIError {
	if err := r.ParseForm(); err != nil {
		return NewInternalError(err)
	}

	problem := r.FormValue("problem")
	if problem == "" {
		return &APIError{
			Code:    http.StatusBadRequest,
			Message: "No problem provided.",
		}
	}

	language := r.FormValue("language")
	if language == "" {
		return &APIError{
			Code:    http.StatusBadRequest,
			Message: "No language provided.",
		}
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return &APIError{
			Code:    http.StatusBadRequest,
			Message: "Please upload a program.",
			Err:     err,
		}
	}

	const fileSizeLimit = 1024
	if header.Size > fileSizeLimit {
		return &APIError{
			Code:    http.StatusBadRequest,
			Message: "File is larger than 1MB.",
		}
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return NewInternalError(err)
	}

	upload := &Upload{
		Problem:  problem,
		Language: language,
		Text:     string(data),
	}

	job, apiErr := a.queueJob(upload)
	if apiErr != nil {
		return apiErr
	}

	if err := json.NewEncoder(w).Encode(job); err != nil {
		return NewInternalError(err)
	}

	return nil
}

func (a *Autograder) UploadProgramText(w http.ResponseWriter, r *http.Request) *APIError {
	upload := new(Upload)
	if err := json.NewDecoder(r.Body).Decode(&upload); err != nil {
		return NewInternalError(err)
	}

	job, apiErr := a.queueJob(upload)
	if apiErr != nil {
		return apiErr
	}

	if err := json.NewEncoder(w).Encode(job); err != nil {
		return NewInternalError(err)
	}

	return nil
}

// Get a running job from its ID.
func (a *Autograder) GetJob(w http.ResponseWriter, r *http.Request) *APIError {
	val := r.URL.Query().Get("id")
	if val == "" {
		return &APIError{
			Code:    http.StatusBadRequest,
			Message: "No ID provided.",
		}
	}

	id, err := strconv.Atoi(val)
	if err != nil {
		return &APIError{
			Code:    http.StatusBadRequest,
			Message: "ID must be an integer.",
			Err:     err,
		}
	}

	if id < 0 || id >= len(a.Jobs) {
		return &APIError{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Job %d does not exist.", id),
		}
	}

	if err := json.NewEncoder(w).Encode(a.Jobs[id]); err != nil {
		return NewInternalError(err)
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

func (a *Autograder) queueJob(upload *Upload) (*Job, *APIError) {
	dir := os.Getenv("PROBLEMS_DIR")
	if dir == "" {
		return nil, NewInternalError(fmt.Errorf(problemsDirErr))
	}

	if upload.Problem == "" {
		return nil, &APIError{
			Code:    http.StatusBadRequest,
			Message: "No problem specified.",
		}
	}

	_, err := os.Stat(filepath.Join(dir, upload.Problem))
	if os.IsNotExist(err) {
		return nil, &APIError{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Problem %s does not exist.", upload.Problem),
		}
	}

	if upload.Language == "" {
		return nil, &APIError{
			Code:    http.StatusBadRequest,
			Message: "No language specified.",
		}
	}

	if _, ok := languageInfo[upload.Language]; !ok {
		return nil, &APIError{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Language %s is not supported.", upload.Language),
		}
	}

	id := len(a.Jobs)

	job := NewJob(id, upload)
	a.Jobs = append(a.Jobs, job)
	a.runningJobs[id] = job

	return job, nil
}
