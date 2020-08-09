package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/docker/docker/client"

	"github.com/brianstrauch/autograder/errors"
)

type Autograder struct {
	jobs   []*Job
	docker *client.Client
}

func NewAutograder() *Autograder {
	docker, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &Autograder{
		docker: docker,
	}
}

// Upload a program for grading as a multipart form
func (a *Autograder) PostProgramFile(w http.ResponseWriter, r *http.Request) *errors.APIError {
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
			Message: "Please program a program.",
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

	program := &Program{
		Problem:  problem,
		Language: language,
		Text:     string(data),
	}

	jobs, apiErr := a.startJobs(program)
	if apiErr != nil {
		return apiErr
	}

	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

// Upload a program for grading in JSON format
func (a *Autograder) PostProgram(w http.ResponseWriter, r *http.Request) *errors.APIError {
	program := new(Program)
	if err := json.NewDecoder(r.Body).Decode(&program); err != nil {
		return errors.NewInternalError(err)
	}

	jobs, apiErr := a.startJobs(program)
	if apiErr != nil {
		return apiErr
	}

	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

// Check the status of a job
func (a *Autograder) GetJob(w http.ResponseWriter, r *http.Request) *errors.APIError {
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

	if id < 0 || id >= len(a.jobs) {
		return &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Job %d does not exist.", id),
		}
	}

	if err := json.NewEncoder(w).Encode(a.jobs[id]); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

// Validate and run a job in a goroutine
func (a *Autograder) startJobs(program *Program) ([]*Job, *errors.APIError) {
	if program.Problem == "" {
		return nil, &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: "No problem specified.",
		}
	}

	dir := filepath.Join(getProblemsDir(), program.Problem)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Problem %s does not exist.", program.Problem),
		}
	}

	if program.Language == "" {
		return nil, &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: "No language specified.",
		}
	}

	if _, ok := languages[program.Language]; !ok {
		return nil, &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Language %s is not supported.", program.Language),
		}
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.in"))
	if err != nil {
		return nil, errors.NewInternalError(err)
	}

	jobs := make([]*Job, len(files))
	for i := range jobs {
		jobs[i] = NewJob(len(a.jobs), program, i)
		a.jobs = append(a.jobs, jobs[i])
		go jobs[i].run(a.docker)
	}

	return jobs, nil
}

func getProblemsDir() string {
	if val, ok := os.LookupEnv("PROBLEMS_DIR"); ok {
		return val
	}
	return "problems"
}
