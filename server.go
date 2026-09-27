// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"html/template"
	"net/http"
)

type version struct {
	SHA   string `json:"sha"`
	Built string `json:"built"`
}

var page = template.Must(template.New("home").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>Patchy preview demo</title></head><body><main><h1>Hello, previews!</h1>
<p>A small, stateless Go app. No accounts, storage, secrets, or outbound requests.</p>
<p>Revision: <code>{{.SHA}}</code></p><p>Built: <code>{{.Built}}</code></p>
<nav><a href="/healthz">Health</a> · <a href="/version">Version JSON</a></nav>
</main></body></html>
`))

func newHandler(sha, built string) http.Handler {
	info := version{SHA: sha, Built: built}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, info)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}
