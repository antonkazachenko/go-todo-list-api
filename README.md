# Go Todo List API

#### English | [Français](README.fr.md) | [Русский](README.ru.md)
### Checkout the [Live Demo](https://go-todo-list-api.onrender.com/) (password: `test12345`)

**Please note that the live demo is hosted on a free Render plan, so it may take some time for the server to start up when accessed.**

## Project Overview
This project is a simple todo list API built in Go. It provides a backend service that allows users to create, read, 
update and delete tasks. The application uses JSON Web Tokens (JWT) for secure authentication and SQLite for persistent 
data storage. In addition to basic task management, the API also supports scheduling tasks with custom repeat intervals.

The API is built using the layered architecture pattern, with separate layers for the API. The are 4 main layers:
- **Controller Layer**: Handles incoming HTTP requests and routes them to the appropriate handler located in the `internal/server` directory.
- **Service Layer**: Contains the business logic for the application located in the `internal/service` directory.
- **Repository Layer**: Handles the interaction with the database located in the `internal/storage` directory.
- **Entities Layer**: Contains the data entities used by the application located in the `internal/models` directory.

In addition, app has a simple web interface to interact with the API. The web interface is built using HTML, CSS, and JavaScript which 
is minified and located in the `web/` directory.

There is a Dockerfile in the project root directory that can be used to build a Docker image of the application. The dockerfile
uses a multi-stage build to create a lightweight image with the application binary and the necessary files.

## Features
- Task management: Create, read, update and delete tasks.
- Task scheduling: Schedule tasks for future dates with the ability to set up a custom repeat interval.
- Authentication: Secure login with JWT.
- Persistent storage using SQLite.
- RESTful API built with the Chi router.

## Dependencies
The project uses the following dependencies:
- **Chi Router**: Lightweight and idiomatic routing in Go (`github.com/go-chi/chi/v5`)
- **JWT**: Handling authentication (`github.com/golang-jwt/jwt/v4`)
- **SQLx**: SQL toolkit for Go (`github.com/jmoiron/sqlx`)
- **SQLite3**: Database driver (`github.com/mattn/go-sqlite3`)
- **Testify**: Testing utilities (`github.com/stretchr/testify`)

**Note:** You need Go version **1.22.2** or higher to run the application.

## Installation
1. Clone the repository:
```bash  
git clone https://github.com/antonkazachenko/go-todo-list-api.git
```
2. Navigate to the project directory:
```bash
cd go-todo-list-api
```
3. Install the dependencies:
```bash
go mod tidy
```

## Environment Variables
To run the server locally, you can set up the following environment variables:

- `TODO_DBFILE`: Path to the SQLite database file (default: `scheduler.db`)
- `TODO_PORT`: Port on which the server will run (default: `7540`)
- `TODO_PASSWORD`: Password used for JWT signing (default: empty)

You can set these environment variables in your shell before running the application:

```bash
export TODO_DBFILE="your_db_file.db"
export TODO_PORT="your_port_number"
export TODO_PASSWORD="your_password"
```

In case if you don't set the environment variables, the application will use the default values.

## Usage
1. Build and run the project:
```bash
go run main.go
```
2. Access the API via `http://localhost:PORT/` (Replace `PORT` with the actual port specified in your configuration or the default port `7540`).

## API Endpoints
Here is a brief overview of the main API endpoints:

- **POST /api/task** - Create a new task.
- **GET /api/tasks** - Get all tasks.
- **GET /api/task** - Get a specific task.
- **PUT /api/task** - Update a specific task.
- **DELETE /api/task** - Delete a specific task.
- **POST /api/task/done** - Mark a task as done.
- **POST /api/signin** - User login.

## Authentication
Authentication in this application is handled using JSON Web Tokens (JWT). Upon successful login, a JWT is generated and 
returned to the user. This functionality can be seen in the network tab in the browser's developer tools.

If you don't set up the `TODO_PASSWORD` environment variable, the application will not use JWT for authentication.


## Testing
- The project uses Testify for unit testing.
- Tests are located in the `tests/` directory.
- Run the tests using:
```bash
go test ./tests
```

## Contributing
Contributions are welcome! Please feel free to submit a Pull Request.

## License
This project is licensed under the MIT License. See the `LICENSE` file for details.
