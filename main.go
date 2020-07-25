package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/brianstrauch/autograder/errors"
)

func main() {
	if err := pullImages(); err != nil {
		panic(err)
	}

	a := NewAutograder()

	routes := map[string]func(w http.ResponseWriter, r *http.Request) *errors.APIError{
		"/text": a.PostProgram,
		"/file": a.PostProgramFile,
		"/job":  a.GetJob,
	}

	for pattern, function := range routes {
		http.Handle(pattern, errors.ErrorHandler(function))
	}

	port := 1024
	if str := os.Getenv("PORT"); str != "" {
		var err error
		port, err = strconv.Atoi(str)
		if err != nil {
			panic(err)
		}
	}

	addr := fmt.Sprintf(":%d", port)
	log.Println("Listening at http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
