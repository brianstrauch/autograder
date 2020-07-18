package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	inputFile  = "in.txt"
	outputFile = "out.txt"
)

type Job struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`

	upload *Upload
}

func NewJob(id int, upload *Upload) *Job {
	return &Job{
		ID:     id,
		Status: "READY",
		upload: upload,
	}
}

// Containerize the program and run. Then, delete the container.
func (j *Job) run(docker *client.Client) {
	ctx := context.Background()

	info := languageInfo[j.upload.Language]
	cfg := &container.Config{
		Image: info.image,
		Cmd:   info.command,
	}

	con, err := docker.ContainerCreate(ctx, cfg, nil, nil, "")
	if err != nil {
		j.fail(err)
		return
	}
	defer cleanup(docker, con.ID)

	var buf bytes.Buffer
	w := tar.NewWriter(&buf)

	if err := writeTAR(w, languageInfo[j.upload.Language].filename, []byte(j.upload.Text)); err != nil {
		j.fail(err)
		return
	}

	dir := os.Getenv("PROBLEMS_DIR")
	if dir == "" {
		j.fail(fmt.Errorf(problemsDirErr))
	}
	path := filepath.Join(dir, j.upload.Problem, inputFile)

	in, err := ioutil.ReadFile(path)
	if err != nil {
		j.fail(err)
		return
	}

	if err := writeTAR(w, inputFile, in); err != nil {
		j.fail(err)
		return
	}

	file := bytes.NewReader(buf.Bytes())

	if err := docker.CopyToContainer(ctx, con.ID, "/", file, types.CopyToContainerOptions{}); err != nil {
		j.fail(err)
		return
	}

	if err := docker.ContainerStart(ctx, con.ID, types.ContainerStartOptions{}); err != nil {
		j.fail(err)
		return
	}

	if _, err := docker.ContainerWait(ctx, con.ID); err != nil {
		j.fail(err)
		return
	}

	out, err := docker.ContainerLogs(ctx, con.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		j.fail(err)
		return
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if _, err := stdcopy.StdCopy(stdout, stderr, out); err != nil {
		j.fail(err)
		return
	}
	j.Stdout = stdout.String()
	j.Stderr = stderr.String()

	j.grade()
}

func (j *Job) fail(err error) {
	log.Printf("JOB %d FAILED: %s\n", j.ID, err)
	j.Status = "ERROR"
}

// Check if output matches the solution file.
func (j *Job) grade() {
	if j.Stderr != "" {
		j.Status = "ERROR"
		return
	}

	dir := os.Getenv("PROBLEMS_DIR")
	if dir == "" {
		j.fail(fmt.Errorf(problemsDirErr))
	}
	dir = filepath.Join(dir, j.upload.Problem, outputFile)

	out, err := ioutil.ReadFile(dir)
	if err != nil {
		j.fail(err)
		return
	}

	if j.Stdout == string(out) {
		j.Status = "RIGHT"
	} else {
		j.Status = "WRONG"
	}
}

func cleanup(docker *client.Client, containerID string) {
	ctx := context.Background()

	if err := docker.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{}); err != nil {
		log.Println(err)
	}
}

// Write a file to a TAR archive
func writeTAR(w *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(data)),
	}

	if err := w.WriteHeader(header); err != nil {
		return err
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}