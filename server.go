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
<title>Patchy preview demo</title>
<style>.card{background-color:#7c3aed;color:#fff;padding:1.5rem;border-radius:0.5rem;font-family:sans-serif}</style></head><body><main class="card"><h1>Hello from patchy</h1>
<p>A small, stateless Go app. No accounts, storage, secrets, or outbound requests.</p>
<p>Revision: <code>{{.SHA}}</code></p><p>Built: <code>{{.Built}}</code></p>
<nav><a href="/healthz">Health</a> · <a href="/version">Version JSON</a></nav>
<footer>Deployed by patchy · built {{.Built}}</footer>
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
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}
