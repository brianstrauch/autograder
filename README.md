# Autograder API

Upload programs for grading in file or data format.
The autograder will run them inside a container and grade their output.

1. Upload a program
    1. `POST /file` (multipart/form-data)
        ```
        <form action="http://localhost:1024/file" method="post" enctype="multipart/form-data">
            <input name="problem" />
            <input name="language" />
            <input name="file" type="file" />
            
            <input type="submit" />
        </form>
        ```
    2. `POST /text` (json)
        ```
        {
            "problem": "hello-world",
            "language": "python",
            "text": "print('Hello, World!')"
        }
        ```
2. Check the status of a program
    * `GET /job?id=0`
    ```
    {
        "id": 0,
        "status": "RIGHT",
        "stdout": "Hello, World!\n",
        "stderr": ""
    }
    ```
