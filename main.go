package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"

	"github.com/brianstrauch/autograder/errors"
)

const defaultPort = 1024

func init() {
	if godotenv.Load() != nil {
		log.Println("No .env file, using default values")
	}
}

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

	port := defaultPort
	if val, ok := os.LookupEnv("PORT"); ok {
		var err error
		port, err = strconv.Atoi(val)
		if err != nil {
			panic(err)
		}
	}

	addr := fmt.Sprintf(":%d", port)
	log.Println("Listening at http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
