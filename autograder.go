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

const (
	inputFile          = "in.txt"
	outputFile         = "out.txt"
	defaultProblemsDir = "problems"
)

type autograder struct {
	jobs   []*Job
	docker *client.Client
}

type Program struct {
	Problem  string `json:"problem"`
	Language string `json:"language"`
	Text     string `json:"text"`
}

func NewAutograder() *autograder {
	docker, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &autograder{
		docker: docker,
	}
}

// Upload a program for grading as a multipart form
func (a *autograder) PostProgramFile(w http.ResponseWriter, r *http.Request) *errors.APIError {
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

	upload := &Program{
		Problem:  problem,
		Language: language,
		Text:     string(data),
	}

	job, apiErr := a.startJob(upload)
	if apiErr != nil {
		return apiErr
	}

	if err := json.NewEncoder(w).Encode(job); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

// Upload a program for grading in JSON format
func (a *autograder) PostProgram(w http.ResponseWriter, r *http.Request) *errors.APIError {
	program := new(Program)
	if err := json.NewDecoder(r.Body).Decode(&program); err != nil {
		return errors.NewInternalError(err)
	}

	job, apiErr := a.startJob(program)
	if apiErr != nil {
		return apiErr
	}

	if err := json.NewEncoder(w).Encode(job); err != nil {
		return errors.NewInternalError(err)
	}

	return nil
}

// Check the status of a job
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
func (a *autograder) startJob(program *Program) (*Job, *errors.APIError) {
	if program.Problem == "" {
		return nil, &errors.APIError{
			Code:    http.StatusBadRequest,
			Message: "No problem specified.",
		}
	}

	dir := defaultProblemsDir
	if val, ok := os.LookupEnv("PROBLEMS_DIR"); ok {
		dir = val
	}

	if _, err := os.Stat(filepath.Join(dir, program.Problem)); os.IsNotExist(err) {
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

	job := NewJob(len(a.jobs), program)
	a.jobs = append(a.jobs, job)
	go job.run(a.docker)

	return job, nil
}
