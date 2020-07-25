# Autograder API

Upload programs for grading. The autograder will run them inside a container and compare their output.

## Local Setup

1. Download [Docker](https://www.docker.com/products/docker-desktop), install, and start
2. Optionally create a `.env` file to set the `PORT` or `PROBLEMS_DIR`
2. Use `make run` to start the API on http://localhost:1024

## Usage

1. Upload a program
    1. `POST /text`
        ```
        {
            "problem": "hello-world",
            "language": "python",
            "text": "print('Hello, World!')"
        }
        ```
    2. `POST /file`
        ```
        <form method="POST" action="http://localhost:1024/file" enctype="multipart/form-data">
            <input name="problem" />
            <input name="language" />
            <input name="file" type="file" />
            
            <input type="submit" />
        </form>
        ```
2. Check the program's status
    1. `GET /job?id=0`
        ```
        {
            "id": 0,
            "status": "RIGHT",
            "stdout": "Hello, World!\n",
            "stderr": ""
        }
        ```
