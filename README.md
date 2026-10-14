# Custom HTTP Server in Go

This project is a custom HTTP server built from scratch in Go. It does not rely on the standard net/http package for the core request handling. Instead, it manually parses HTTP/1.1 requests, matches routes, applies middleware, and writes responses.

## Features

- Manual HTTP request parsing
- Custom router with route registration
- Middleware support for logging, request IDs, metrics, and recovery
- Response writer for plain text and JSON responses
- Graceful shutdown handling
- Basic HTTP/1.1 response behavior

## Project structure

- cmd/server/main.go: server entry point and main accept loop
- parser: request parsing logic and tests
- router: request routing and handler dispatch
- response: response creation and output
- middleware: logging, request IDs, metrics, and recovery middleware
- server: server package placeholder

## Running the server

1. Open a terminal in the project root.
2. Run:

   go run ./cmd/server

3. The server listens on port 8080.

4. Test it with:

   curl http://localhost:8080/
   curl http://localhost:8080/health
   curl http://localhost:8080/about

## Example routes

- GET / returns a welcome message
- GET /health returns OK
- GET /about returns a JSON response

## Notes

This repository is designed to help understand how HTTP communication works under the hood, rather than just using a framework or a ready-made server implementation.
