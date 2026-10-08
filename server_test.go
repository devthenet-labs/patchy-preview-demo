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
	"time"
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
	if !strings.Contains(body, "Hello from patchy") || !strings.Contains(body, "#7c3aed") {
		t.Fatalf("body=%q", body)
	}
}

func TestFooter(t *testing.T) {
	h := newHandler("test", "test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	if !strings.Contains(body, "Deployed by patchy") {
		t.Fatalf("body=%q", body)
	}
	if strings.Index(body, "Built:") > strings.Index(body, "Deployed by patchy") {
		t.Fatalf("footer must appear after build-time marker: body=%q", body)
	}
	if !strings.Contains(body, "Deployed by patchy · built test") {
		t.Fatalf("footer missing build time: body=%q", body)
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
	headingIdx := strings.Index(body, "Hello from patchy")
	subtitleIdx := strings.Index(body, subtitle)
	bodyParaIdx := strings.Index(body, "A small, stateless Go app")
	if headingIdx < 0 || subtitleIdx < 0 || bodyParaIdx < 0 || !(headingIdx < subtitleIdx && subtitleIdx < bodyParaIdx) {
		t.Fatalf("subtitle must render directly under the heading and before the body paragraph: body=%q", body)
	}
	if !strings.Contains(body, `class="subtitle"`) {
		t.Fatalf("subtitle paragraph must use the subtitle style: body=%q", body)
	}
}

func TestRelativeTime(t *testing.T) {
	for _, tc := range []struct {
		since time.Duration
		want  string
	}{
		{0, "just now"},
		{30 * time.Second, "just now"},
		{-5 * time.Second, "just now"},
		{1 * time.Minute, "1 minute ago"},
		{90 * time.Second, "1 minute ago"},
		{2 * time.Minute, "2 minutes ago"},
		{47 * time.Minute, "47 minutes ago"},
	} {
		if got := relativeTime(tc.since); got != tc.want {
			t.Errorf("relativeTime(%v) = %q, want %q", tc.since, got, tc.want)
		}
	}
}

func TestDeployedLine(t *testing.T) {
	for _, tc := range []struct {
		name   string
		offset time.Duration
		want   string
	}{
		{"now", 0, "Deployed just now"},
		{"over a minute", -70 * time.Second, "Deployed 1 minute ago"},
		{"ten minutes", -10 * time.Minute, "Deployed 10 minutes ago"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			built := time.Now().Add(tc.offset).Format(time.RFC3339)
			h := newHandler("sha", built)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
			body := w.Body.String()
			if !strings.Contains(body, tc.want) {
				t.Fatalf("body=%q, want containing %q", body, tc.want)
			}
		})
	}
}

func TestDeployedLineOmittedWhenUnparseable(t *testing.T) {
	h := newHandler("test", "test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	if strings.Contains(body, "ago") || strings.Contains(body, "Deployed just now") {
		t.Fatalf("expected no deployed line for unparseable build time: body=%q", body)
	}
}

func TestBadge(t *testing.T) {
	h := newHandler("test", "test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	body := w.Body.String()
	if !strings.Contains(body, `class="badge"`) || !strings.Contains(body, "Preview") {
		t.Fatalf("missing badge: body=%q", body)
	}
	headingIdx := strings.Index(body, "Hello from patchy")
	badgeIdx := strings.Index(body, "Preview")
	subtitleIdx := strings.Index(body, "Previewed by patchy")
	if headingIdx < 0 || badgeIdx < 0 || subtitleIdx < 0 || !(headingIdx < badgeIdx && badgeIdx < subtitleIdx) {
		t.Fatalf("badge must render inside the heading: body=%q", body)
	}
	if !strings.Contains(body, "#f59e0b") || !strings.Contains(body, "#7c3aed") {
		t.Fatalf("badge colour must contrast with card colour: body=%q", body)
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
