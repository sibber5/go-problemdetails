// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025 sibber (GitHub: sibber5)

package problemdetails

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
)

// Recoverer is a middleware that recovers from panics and returns a HTTP 500 (Internal Server Error) problem details response, if possible.
// The details field of the problem details response contains the panic message and, if stackFrameIdx >= 0, the stackFrameIdx'th caller in the stack frame.
//
// stackFrameIdx: The index of the caller in the stack frame to include in the details field in the response body.
// If < 0 then it wond be included. Note that the actual index used is actually stackFrameIdx + 3 in order to skip the frames for this middleware and runtime/panic.go.
func Recoverer(stackFrameIdx int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Based on original work from https://github.com/go-chi/chi/blob/9b9fb55def404397748a9fc7e044efe9db1d618e/middleware/recoverer.go
			// Licensed under the MIT License: https://github.com/go-chi/chi/blob/9b9fb55def404397748a9fc7e044efe9db1d618e/LICENSE
			// Copyright (c) 2015-present Peter Kieltyka (https://github.com/pkieltyka), Google Inc.
			defer func() {
				if rec := recover(); rec != nil {
					if rec == http.ErrAbortHandler {
						// We don't recover http.ErrAbortHandler so that the response to the client is aborted, this should not be logged.
						panic(rec)
					}

					if r.Header.Get("Connection") == "Upgrade" {
						return
					}

					var detail string
					if stackFrameIdx >= 0 {
						var buf [1]uintptr
						pc := buf[:]
						n := runtime.Callers(stackFrameIdx+3, pc) // Skip 3 frames for this middleware + runtime/panic.go.
						if n == 1 {
							frame, _ := runtime.CallersFrames(pc).Next()
							detail = fmt.Sprintf("panic: '%v' at %s:%d", rec, frame.File, frame.Line)
						}
					}
					if detail == "" {
						detail = fmt.Sprintf("panic: '%v'", rec)
					}

					Write(w, r, http.StatusInternalServerError, detail, "")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

type ctxKey string

// Value: `*problemdetails.Context`
var CtxKey = ctxKey("problemdetails")

type Context struct {
	pd *ProblemDetails
}

// Details returns the problem details object written to the current response body if one was written, otherwise nil.
func (c *Context) Details() *ProblemDetails {
	return c.pd
}

// ProblemDetailsContext is a middleware that injects a `*problemdetails.Context` object with key `problemdetails.CtxKey` into
// the context of each request.
//
// The `*problemdetails.Context` is meant to be used *only after* the request handler has run (for e.g. request logging).
// The `Details()` method on it will return a `*ProblemDetails` to the same object that was written to the response body,
// if `problemdetails.Write` was called, otherwise `nil`.
func ProblemDetailsContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), CtxKey, &Context{})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ProblemDetailsConverter returns a middleware that intercepts HTTP responses with status codes >= 400
// and converts them to RFC 9457 compliant problem detail responses if they are not already
// (by checking if the Content-Type starts with "application/problem+json").
//
// logCallback: a function to be called with the request and status code when an error response is intercepted and converted.
func ProblemDetailsConverter(logCallback func(r *http.Request, status int)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ri := interceptorPool.Get().(*responseInterceptor)
			ri.ResponseWriter = w
			ri.status = 0 // 0 indicates WriteHeader has not been called.
			ri.bodyWritten = false
			defer interceptorPool.Put(ri)

			next.ServeHTTP(ri, r)

			ri.ResponseWriter = nil

			if ri.status >= 400 && !ri.bodyWritten && !strings.HasPrefix(w.Header().Get("Content-Type"), "application/problem+json") {
				w.Header().Del("Content-Encoding")
				w.Header().Del("Vary")
				w.Header().Del("Content-Length")

				Write(w, r, ri.status, "", "")

				logCallback(r, ri.status)
				return
			}

			// If we didn't convert the response, ensure the status header is written
			// in cases where only WriteHeader was called, like with 204.
			if !ri.bodyWritten && ri.status != 0 {
				w.WriteHeader(ri.status)
			}
		})
	}
}

var interceptorPool = sync.Pool{
	New: func() any {
		return &responseInterceptor{}
	},
}

type responseInterceptor struct {
	http.ResponseWriter
	status      int
	bodyWritten bool
}

func (ri *responseInterceptor) WriteHeader(status int) {
	ri.status = status
}

func (ri *responseInterceptor) Write(body []byte) (int, error) {
	if ri.status >= 400 && len(body) == 0 {
		return 0, nil
	}
	if !ri.bodyWritten { // handle things like maybeWriteHeader() in wrap_writer.go in github.com/go-chi/chi/v5@v5.2.2/middleware/wrap_writer.go:116
		if ri.status == 0 {
			ri.status = http.StatusOK
		}
		ri.ResponseWriter.WriteHeader(ri.status)
	}
	ri.bodyWritten = true
	return ri.ResponseWriter.Write(body)
}
