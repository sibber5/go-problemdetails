# go-problemdetails

[![Version](https://img.shields.io/github/v/tag/sibber5/go-problemdetails?include_prereleases&label=latest)](https://pkg.go.dev/github.com/sibber5/go-problemdetails)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue.svg?color=lightgrey)](LICENSE)

A zero 3P dependency Go library for writing [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457.html) compliant HTTP Problem Details responses, with middleware for error conversion and panic recovery.

## Features

- Write [Problem Details](https://www.rfc-editor.org/rfc/rfc9457.html) responses with additional fields for traceability and aggregate errors.
- Middleware for:
  - Converting plain error responses (status >= 400) to Problem Details.
  - Recovering from panics and returning Problem Details responses.
  - Injecting problem details context into requests for failed requests.

## Installation

```sh
go get github.com/sibber5/go-problemdetails/problemdetails
```

## Usage

### Writing Problem Details

```go
import (
    "net/http"
    "github.com/sibber5/go-problemdetails/problemdetails"
)

func handler(w http.ResponseWriter, r *http.Request) {
    problemdetails.Write(w, r, http.StatusBadRequest, "Invalid input", errcodes.InvalidInput)
}
```

### Middleware

#### ProblemDetailsConverter

Converts all error responses (status >= 400) to Problem Details if not already (checks via Content-Type).

```go
problemdetails.ProblemDetailsConverter(func(r *http.Request, status int) {
    // This runs when an error response is intercepted and converted.
    slog.InfoContext(r.Context(), "Intercepted error response", "status", status)
})
```

This middleware - like request loggers for example - processes *after* it calls `next.ServeHTTP`, meaning the earlier you register it, the later it runs.  
It must be registered as early as possible, after middlewares that inject context like request IDs, and before any other middleware that also runs after serving, including request loggers.

#### Recoverer

Recovers from panics and writes a Problem Details response.  
It should be registered as early as possible.

#### ProblemDetailsContext

Injects a context object to retrieve the problem details written to the response for failed requests.

#### Example of middleware order

```go
r.Use(middleware.RequestID)
r.Use(problemdetails.ProblemDetailsContext)

// Should be before the request logger to let the request logger log the panic. skip 3 frames for the request logger.
// You can register the recoverer after the ProblemDetailsConverter. It doesn't matter since the recoverer already write problem details responses so the ProblemDetailsConverter will not intercept them.
r.Use(problemdetails.Recoverer(3))

// Since this wraps (processes after it calls next.ServeHTTP), it will actually run *after* anything below it.
r.Use(problemdetails.ProblemDetailsConverter(func(r *http.Request, status int) {
    slog.InfoContext(r.Context(), "Intercepted non-problem details error response", "status", status, "path", getRequestPath(r))
}))

// The request logger can catch panics to log them, and re-panic for problem details recovrer to catch.
r.Use(requestLogger)
```

### Configuration

You can set a new default writer in order to configure it.  
For example to write the request ID in the problem details responses:

```go
pdw := &problemdetails.Writer{
    GetRequestID: func(r *http.Request) string { return r.Context().Value(RequestIDCtxKey).(string) },
}
problemdetails.SetDefault(pdw)
```

## License

This project is licensed under the BSD 3-Clause "New" or "Revised" License - see the [LICENSE](LICENSE) file for details.
