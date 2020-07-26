package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGradeRight(t *testing.T) {
	dir := mockProblemsDir(t, "hello-world", "Hello, World!")
	defer os.RemoveAll(dir)

	j := &Job{
		program: &Program{
			Problem: "hello-world",
		},
		Stdout: "Hello, World!",
	}
	j.grade()

	require.Equal(t, j.Status, "RIGHT")
}

func TestGradeWrong(t *testing.T) {
	dir := mockProblemsDir(t, "hello-world", "Hello, World!")
	defer os.RemoveAll(dir)

	j := &Job{
		program: &Program{
			Problem: "hello-world",
		},
		Stdout: "Hello, World.",
	}
	j.grade()

	require.Equal(t, j.Status, "WRONG")
}

func mockProblemsDir(t *testing.T, problem, output string) string {
	dir := os.TempDir()
	require.NoError(t, os.Setenv("PROBLEMS_DIR", dir))

	problemDir := filepath.Join(dir, problem)
	require.NoError(t, os.MkdirAll(problemDir, 0777))

	file := filepath.Join(problemDir, outputFile)
	require.NoError(t, ioutil.WriteFile(file, []byte(output), 0644))

	return dir
}

func TestGradeProgramError(t *testing.T) {
	j := &Job{Stderr: "error"}
	j.grade()
	require.Equal(t, j.Status, "ERROR")
}
