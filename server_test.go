// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRoutes(t *testing.T) {
	h := newHandler("0123456789abcdef", "2026-09-27T00:00:00Z")
	for _, tc := range []struct {
		method, path, contentType, contains string
		status                              int
	}{
		{"GET", "/", "text/html; charset=utf-8", "Hello From Patchy", 200},
		{"GET", "/healthz", "application/json", `{"status":"ok"}`, 200},
		{"GET", "/version", "application/json", `"sha":"0123456789abcdef"`, 200},
		{"GET", "/id", "application/json", `"id":"`, 200},
		{"GET", "/semver?v=v1.2.3", "application/json", `"valid":true`, 200},
		{"GET", "/wordcount?text=hello+world", "application/json", `"words":2`, 200},
		{"GET", "/wordcount", "application/json", `"error"`, 400},
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

func TestID(t *testing.T) {
	h := newHandler("test", "test")
	call := func() string {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/id", nil))
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("response: status=%d type=%q", w.Code, w.Header().Get("Content-Type"))
		}
		var got idResponse
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v, body=%q", err, w.Body.String())
		}
		if _, err := uuid.Parse(got.ID); err != nil {
			t.Fatalf("not a valid uuid: %q: %v", got.ID, err)
		}
		return got.ID
	}
	first, second := call(), call()
	if first == second {
		t.Fatalf("expected two different ids, got %q twice", first)
	}
}

func TestSemver(t *testing.T) {
	h := newHandler("test", "test")
	call := func(query string) semverResponse {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/semver"+query, nil))
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("response: status=%d type=%q", w.Code, w.Header().Get("Content-Type"))
		}
		var got semverResponse
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v, body=%q", err, w.Body.String())
		}
		return got
	}
	for _, tc := range []struct {
		query string
		want  semverResponse
	}{
		{"?v=v1.2.3", semverResponse{Valid: true, Canonical: "v1.2.3"}},
		{"?v=v1.2", semverResponse{Valid: true, Canonical: "v1.2.0"}},
		{"?v=1.2.3", semverResponse{Valid: false, Canonical: ""}},
		{"", semverResponse{Valid: false, Canonical: ""}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			if got := call(tc.query); got != tc.want {
				t.Fatalf("got=%+v, want=%+v", got, tc.want)
			}
		})
	}
}

func TestSlug(t *testing.T) {
	h := newHandler("test", "test")
	call := func(query string) slugResponse {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/slug"+query, nil))
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("response: status=%d type=%q", w.Code, w.Header().Get("Content-Type"))
		}
		var got slugResponse
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v, body=%q", err, w.Body.String())
		}
		return got
	}
	for _, tc := range []struct {
		query string
		want  slugResponse
	}{
		{"?s=Hello+World", slugResponse{Slug: "hello-world"}},
		{"?s=Hello%20World%21", slugResponse{Slug: "hello-world"}},
		{"?s=Already-A-Slug", slugResponse{Slug: "already-a-slug"}},
		{"?s=", slugResponse{Slug: ""}},
		{"", slugResponse{Slug: ""}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			if got := call(tc.query); got != tc.want {
				t.Fatalf("got=%+v, want=%+v", got, tc.want)
			}
		})
	}
}

func TestWordCount(t *testing.T) {
	h := newHandler("test", "test")
	call := func(query string) wordCountResponse {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/wordcount"+query, nil))
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("response: status=%d type=%q", w.Code, w.Header().Get("Content-Type"))
		}
		var got wordCountResponse
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v, body=%q", err, w.Body.String())
		}
		return got
	}
	for _, tc := range []struct {
		query string
		want  wordCountResponse
	}{
		{"?text=", wordCountResponse{Words: 0, Characters: 0}},
		{"?text=hello", wordCountResponse{Words: 1, Characters: 5}},
		{"?text=hello+world", wordCountResponse{Words: 2, Characters: 11}},
		{"?text=hello%20%20%20world", wordCountResponse{Words: 2, Characters: 13}},
		{"?text=h%C3%A9llo", wordCountResponse{Words: 1, Characters: 5}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			if got := call(tc.query); got != tc.want {
				t.Fatalf("got=%+v, want=%+v", got, tc.want)
			}
		})
	}
}

func TestWordCountMissingText(t *testing.T) {
	h := newHandler("test", "test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/wordcount", nil))
	if w.Code != http.StatusBadRequest || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("response: status=%d type=%q", w.Code, w.Header().Get("Content-Type"))
	}
	var got errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.Error == "" {
		t.Fatalf("unmarshal: %v, body=%q", err, w.Body.String())
	}
}

func TestCardColour(t *testing.T) {
	h := newHandler("test", "test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	if !strings.Contains(body, "Hello From Patchy") || !strings.Contains(body, "#7c3aed") {
		t.Fatalf("body=%q", body)
	}
}

func TestFooter(t *testing.T) {
	h := newHandler("abc123def", "test-build")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	year := time.Now().UTC().Year()
	want := fmt.Sprintf(`<footer class="footer" aria-label="app version">preview-demo abc123def · %d · signed</footer>`, year)
	if !strings.Contains(body, want) {
		t.Fatalf("footer missing or wrong: body=%q", body)
	}
	if !strings.Contains(body, ".footer{color:#666;font-size:0.8rem}") {
		t.Fatalf("footer style missing or wrong: body=%q", body)
	}
}

func TestFooterYear(t *testing.T) {
	h := newHandler("abc123def", "test-build")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	want := fmt.Sprintf("· %d · signed</footer>", time.Now().UTC().Year())
	if !strings.Contains(body, want) {
		t.Fatalf("footer missing current year: body=%q", body)
	}
}

func TestFooterSigned(t *testing.T) {
	h := newHandler("abc123def", "test-build")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	year := time.Now().UTC().Year()
	want := fmt.Sprintf(`preview-demo abc123def · %d · signed</footer>`, year)
	if !strings.Contains(body, want) {
		t.Fatalf("footer missing signed suffix: body=%q", body)
	}
}

func TestSubtitle(t *testing.T) {
	h := newHandler("test", "test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	const subtitle = "Previewed by patchy, one pull request at a time."
	if !strings.Contains(body, subtitle) {
		t.Fatalf("missing subtitle: body=%q", body)
	}
	headingIdx := strings.Index(body, "Hello From Patchy")
	subtitleIdx := strings.Index(body, subtitle)
	bodyParaIdx := strings.Index(body, "A small, stateless Go app")
	if headingIdx < 0 || subtitleIdx < 0 || bodyParaIdx < 0 || !(headingIdx < subtitleIdx && subtitleIdx < bodyParaIdx) {
		t.Fatalf("subtitle must render directly under the heading and before the body paragraph: body=%q", body)
	}
	if !strings.Contains(body, `class="subtitle"`) {
		t.Fatalf("subtitle paragraph must use the subtitle style: body=%q", body)
	}
}

func TestGreetingTitleCase(t *testing.T) {
	h := newHandler("test", "test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	if !strings.Contains(body, "<h1>Hello From Patchy</h1>") {
		t.Fatalf("greeting not title-cased: body=%q", body)
	}
	if strings.Contains(body, "Hello from patchy") {
		t.Fatalf("expected title-cased greeting, found original casing: body=%q", body)
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
