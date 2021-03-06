package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Job struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`

	program    *Program
	inputFile  string
	outputFile string
}

func NewJob(id int, program *Program, part int) *Job {
	return &Job{
		ID:     id,
		Status: "READY",

		program:    program,
		inputFile:  fmt.Sprintf("%d.in", part),
		outputFile: fmt.Sprintf("%d.out", part),
	}
}

// Containerize the program and run. Then, delete the container.
func (j *Job) run(docker *client.Client) {
	info := languages[j.program.Language]

	ctx := context.Background()

	config := &container.Config{
		Image:       info.image,
		Cmd:         info.command,
		AttachStdin: true,
		OpenStdin:   true,
	}
	con, err := docker.ContainerCreate(ctx, config, nil, nil, "")
	if err != nil {
		j.fail(err)
		return
	}
	defer cleanup(docker, con.ID)

	var buf bytes.Buffer
	w := tar.NewWriter(&buf)

	if err := writeTAR(w, languages[j.program.Language].filename, []byte(j.program.Text)); err != nil {
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

	res, err := docker.ContainerAttach(ctx, con.ID, types.ContainerAttachOptions{Stdin: true, Stream: true})
	if err != nil {
		j.fail(err)
		return
	}
	defer res.Close()

	in, err := ioutil.ReadFile(filepath.Join(getProblemsDir(), j.program.Problem, j.inputFile))
	if err != nil {
		j.fail(err)
		return
	}
	if _, err := res.Conn.Write(in); err != nil {
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

// Should never happen, but if it does, exit gracefully.
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

	out, err := ioutil.ReadFile(filepath.Join(getProblemsDir(), j.program.Problem, j.outputFile))
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

// Remove the old container
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
