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
    // Log conversion
})
```

#### Recoverer

Recovers from panics and writes a Problem Details response.

#### ProblemDetailsContext

Injects a context object to retrieve the problem details written to the response for failed requests.

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
