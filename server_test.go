// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	h := newHandler("0123456789abcdef", "2026-09-27T00:00:00Z")
	for _, tc := range []struct {
		method, path, contentType, contains string
		status                              int
	}{
		{"GET", "/", "text/html; charset=utf-8", "Hello from patchy", 200},
		{"GET", "/healthz", "application/json", `{"status":"ok"}`, 200},
		{"GET", "/version", "application/json", `"sha":"0123456789abcdef"`, 200},
		{"GET", "/missing", "text/plain; charset=utf-8", "404", 404},
		{"POST", "/healthz", "text/plain; charset=utf-8", "Method Not Allowed", 405},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status || w.Header().Get("Content-Type") != tc.contentType || !strings.Contains(w.Body.String(), tc.contains) {
				t.Fatalf("response: status=%d type=%q body=%q", w.Code, w.Header().Get("Content-Type"), w.Body.String())
			}
			if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("missing security headers")
			}
			if w.Header().Get("Set-Cookie") != "" {
				t.Fatal("demo must not set cookies")
			}
		})
	}
}

func TestVersionAndEscaping(t *testing.T) {
	const sha = `<script>alert("no")</script>`
	h := newHandler(sha, "test-build")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/version", nil))
	var got version
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got != (version{SHA: sha, Built: "test-build"}) {
		t.Fatalf("version=%+v, error=%v", got, err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if strings.Contains(w.Body.String(), "<script>") || !strings.Contains(w.Body.String(), "&lt;script&gt;") {
		t.Fatal("metadata was not HTML-escaped")
	}
}

func TestCardColour(t *testing.T) {
	h := newHandler("test", "test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	if !strings.Contains(body, "Hello from patchy") || !strings.Contains(body, "#0d9488") {
		t.Fatalf("body=%q", body)
	}
}

func TestHEAD(t *testing.T) {
	server := httptest.NewServer(newHandler("test", "test"))
	t.Cleanup(server.Close)
	for _, path := range []string{"/", "/healthz", "/version"} {
		response, err := server.Client().Head(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if readErr != nil || response.StatusCode != http.StatusOK || len(body) != 0 {
			t.Fatalf("HEAD %s: status=%d bytes=%d error=%v", path, response.StatusCode, len(body), readErr)
		}
	}
}
