package main

import (
	"encoding/json"
	"fmt"
	"github.com/brianstrauch/autograder/errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/docker/client"
)

type autograder struct {
	Jobs        []*Job
	runningJobs map[int]*Job
}

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

type Upload struct {
	Problem  string `json:"problem"`
	Language string `json:"language"`
	Text     string `json:"text"`
}

func NewAutograder() *autograder {
	return &autograder{
		runningJobs: make(map[int]*Job),
	}
}

// Upload a program file, create a job, and add to queue.
func (a *autograder) UploadProgramFile(w http.ResponseWriter, r *http.Request) *errors.APIError {
	if err := r.ParseForm(); err != nil {
		return errors.NewInternalError(err)
	}

	problem := r.FormValue("problem")
	if problem == "" {
		return &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: "No problem provided.",
		}
	}

	language := r.FormValue("language")
	if language == "" {
		return &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: "No language provided.",
		}
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: "Please upload a program.",
			Err:     err,
		}
	}

	const fileSizeLimit = 1024
	if header.Size > fileSizeLimit {
		return &errors.APIError{
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

	job, apiErr := a.queueJob(upload)
	if apiErr != nil {
		return apiErr
	}

	if err := json.NewEncoder(w).Encode(job); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

func (a *autograder) UploadProgramText(w http.ResponseWriter, r *http.Request) *errors.APIError {
	upload := new(Upload)
	if err := json.NewDecoder(r.Body).Decode(&upload); err != nil {
		return errors.NewInternalError(err)
	}

	job, apiErr := a.queueJob(upload)
	if apiErr != nil {
		return apiErr
	}

	if err := json.NewEncoder(w).Encode(job); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

// Get a running job from its ID.
func (a *autograder) GetJob(w http.ResponseWriter, r *http.Request) *errors.APIError {
	val := r.URL.Query().Get("id")
	if val == "" {
		return &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: "No ID provided.",
		}
	}

	id, err := strconv.Atoi(val)
	if err != nil {
		return &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: "ID must be an integer.",
			Err:     err,
		}
	}

	if id < 0 || id >= len(a.Jobs) {
		return &errors.APIError{
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
func (a *autograder) ManageJobs() {
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

func (a *autograder) queueJob(upload *Upload) (*Job, *errors.APIError) {
	dir := os.Getenv("PROBLEMS_DIR")
	if dir == "" {
		return nil, errors.NewInternalError(fmt.Errorf(errors.ProblemsDirErr))
	}

	if upload.Problem == "" {
		return nil, &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: "No problem specified.",
		}
	}

	_, err := os.Stat(filepath.Join(dir, upload.Problem))
	if os.IsNotExist(err) {
		return nil, &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Problem %s does not exist.", upload.Problem),
		}
	}

	if upload.Language == "" {
		return nil, &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: "No language specified.",
		}
	}

	if _, ok := languageInfo[upload.Language]; !ok {
		return nil, &errors.APIError{
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
