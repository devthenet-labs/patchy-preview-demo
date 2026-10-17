// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"golang.org/x/mod/semver"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const greeting = "Hello from patchy"

var greetingTitle = cases.Title(language.English).String(greeting)

type version struct {
	SHA   string `json:"sha"`
	Built string `json:"built"`
}

type idResponse struct {
	ID string `json:"id"`
}

type semverResponse struct {
	Valid     bool   `json:"valid"`
	Canonical string `json:"canonical"`
}

type slugResponse struct {
	Slug string `json:"slug"`
}

type wordCountResponse struct {
	Words      int `json:"words"`
	Characters int `json:"characters"`
}

type errorResponse struct {
	Error string `json:"error"`
}

var page = template.Must(template.New("home").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>Patchy preview demo</title>
<style>.card{background-color:#7c3aed;color:#fff;padding:1.5rem;border-radius:0.5rem;font-family:sans-serif}.subtitle{font-size:0.85rem;opacity:.85;margin:.15rem 0 1rem}.footer{color:#666;font-size:0.8rem}</style></head><body><main class="card"><h1>{{.Greeting}}</h1>
<p class="subtitle">Previewed by patchy, one pull request at a time.</p>
<p>A small, stateless Go app. No accounts, storage, secrets, or outbound requests.</p>
<p>Revision: <code>{{.SHA}}</code></p><p>Built: <code>{{.Built}}</code></p>
<nav><a href="/healthz">Health</a> · <a href="/version">Version JSON</a></nav>
<footer class="footer" aria-label="app version">preview-demo {{.SHA}} · {{.Year}} · signed</footer>
</main></body></html>
`))

func newHandler(sha, built string) http.Handler {
	info := version{SHA: sha, Built: built}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct {
			version
			Year     int
			Greeting string
		}{info, time.Now().UTC().Year(), greetingTitle}
		_ = page.Execute(w, data)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
	})
	mux.HandleFunc("GET /id", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(idResponse{ID: uuid.New().String()})
	})
	mux.HandleFunc("GET /semver", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		v := r.URL.Query().Get("v")
		resp := semverResponse{Valid: semver.IsValid(v)}
		if resp.Valid {
			resp.Canonical = semver.Canonical(v)
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("GET /slug", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		s := r.URL.Query().Get("s")
		_ = json.NewEncoder(w).Encode(slugResponse{Slug: slug.Make(s)})
	})
	mux.HandleFunc("GET /wordcount", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !r.URL.Query().Has("text") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(errorResponse{Error: "missing text parameter"})
			return
		}
		text := r.URL.Query().Get("text")
		_ = json.NewEncoder(w).Encode(wordCountResponse{
			Words:      len(strings.Fields(text)),
			Characters: utf8.RuneCountInString(text),
		})
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}
