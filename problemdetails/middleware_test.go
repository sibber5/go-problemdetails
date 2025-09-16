// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2025 sibber (GitHub: sibber5)
//
// Portions of this file are from https://github.com/go-chi/chi/blob/9b9fb55def404397748a9fc7e044efe9db1d618e/middleware/recoverer_test.go
// Licensed under the MIT License: https://github.com/go-chi/chi/blob/9b9fb55def404397748a9fc7e044efe9db1d618e/LICENSE
// Copyright (c) 2015-present Peter Kieltyka (https://github.com/pkieltyka), Google Inc.

package problemdetails

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

const panicMessage = "foo"

func panickingHandler(http.ResponseWriter, *http.Request) { panic(panicMessage) }

func TestRecoverer(t *testing.T) {
	r := chi.NewRouter()

	r.Use(Recoverer(0))
	r.Get("/", panickingHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	res, resBody := testRequest(t, ts, "GET", "/", nil)
	assertEqual(t, res.StatusCode, http.StatusInternalServerError)

	pd := &ProblemDetails{}
	if err := json.Unmarshal([]byte(resBody), pd); err == nil {
		assertEqual(t, pd.Status, http.StatusInternalServerError)

		_, file, _, ok := runtime.Caller(0)
		if !ok {
			t.Fatal("could not get current file path")
		}
		detail := fmt.Sprintf("panic: '%s' at %s:", panicMessage, file)
		if !strings.HasPrefix(pd.Detail, detail) {
			t.Fatal("unexpected value for ProblemDetails.detail: " + pd.Detail)
		}
	} else {
		t.Fatal(err)
	}
}

func TestRecovererWithoutCaller(t *testing.T) {
	r := chi.NewRouter()

	r.Use(Recoverer(-1))
	r.Get("/", panickingHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	res, resBody := testRequest(t, ts, "GET", "/", nil)
	assertEqual(t, res.StatusCode, http.StatusInternalServerError)

	pd := &ProblemDetails{}
	if err := json.Unmarshal([]byte(resBody), pd); err == nil {
		assertEqual(t, pd.Status, http.StatusInternalServerError)
		assertEqual(t, pd.Detail, fmt.Sprintf("panic: '%v'", panicMessage))
	} else {
		t.Fatal(err)
	}
}

func TestRecovererAbortHandler(t *testing.T) {
	defer func() {
		rcv := recover()
		if rcv != http.ErrAbortHandler {
			t.Fatalf("http.ErrAbortHandler should not be recovered")
		}
	}()

	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(Recoverer(0))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	r.ServeHTTP(w, req)
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	defer resp.Body.Close()

	return resp, string(respBody)
}

func assertEqual(t *testing.T, a, b any) {
	t.Helper()
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("expecting values to be equal but got: '%v' and '%v'", a, b)
	}
}
