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

func init() {
	if godotenv.Load() != nil {
		log.Println("Did not find .env file, using default values.")
	}
}

func main() {
	if err := pullImages(); err != nil {
		panic(err)
	}

	a := NewAutograder()

	routes := map[string]func(w http.ResponseWriter, r *http.Request) *errors.APIError{
		"/upload": a.PostProgram,
		"/job":  a.GetJob,
	}

	for pattern, function := range routes {
		http.Handle(pattern, errors.ErrorHandler(function))
	}

	port := 1024
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
