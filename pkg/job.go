package pkg

import (
	"archive/tar"
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const filename = "program"

type Job struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`

	upload      *Upload
	containerID string
}

func NewJob(id int, upload *Upload) *Job {
	return &Job{
		ID:     id,
		Status: "READY",
		upload: upload,
	}
}

// Containerize the program and run. When complete, delete the container.
func (j *Job) run(docker *client.Client) {
	ctx := context.Background()

	con, err := docker.ContainerCreate(
		ctx,
		&container.Config{
			Image: j.upload.Language,
			Cmd:   []string{j.upload.Language, filename},
		}, nil, nil, "")
	if err != nil {
		j.fail(err)
		return
	}
	defer j.cleanup(docker)
	j.containerID = con.ID

	var buf bytes.Buffer
	w := tar.NewWriter(&buf)

	if err := writeTAR(w, filename, []byte(j.upload.Text)); err != nil {
		j.fail(err)
		return
	}
	file := bytes.NewReader(buf.Bytes())

	if err := docker.CopyToContainer(ctx, j.containerID, "/", file, types.CopyToContainerOptions{}); err != nil {
		j.fail(err)
		return
	}

	if err := docker.ContainerStart(ctx, j.containerID, types.ContainerStartOptions{}); err != nil {
		j.fail(err)
		return
	}

	if _, err := docker.ContainerWait(ctx, j.containerID); err != nil {
		j.fail(err)
		return
	}

	out, err := docker.ContainerLogs(ctx, j.containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
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

	file := filepath.Join("problems", j.upload.Problem, "output.txt")
	out, err := ioutil.ReadFile(file)
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

func (j *Job) cleanup(docker *client.Client) {
	ctx := context.Background()

	if err := docker.ContainerRemove(ctx, j.containerID, types.ContainerRemoveOptions{}); err != nil {
		log.Println(err)
	}
}

// Create a TAR archive with one file.
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
