run: build
	bin/autograder

build:
	go build -o bin/autograder

clean:
	rm -r bin