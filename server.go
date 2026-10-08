// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

type version struct {
	SHA   string `json:"sha"`
	Built string `json:"built"`
}

type pageData struct {
	SHA      string
	Built    string
	Deployed string
}

var page = template.Must(template.New("home").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>Patchy preview demo</title>
<style>.card{background-color:#7c3aed;color:#fff;padding:1.5rem;border-radius:0.5rem;font-family:sans-serif}.subtitle{font-size:0.85rem;opacity:.85;margin:.15rem 0 1rem}.badge{background-color:#f59e0b;color:#1f2937;padding:.15rem .6rem;border-radius:9999px;font-size:.7rem;font-weight:600;vertical-align:middle;margin-left:.5rem}</style></head><body><main class="card"><h1>Hello from patchy <span class="badge">Preview</span></h1>
<p class="subtitle">Previewed by patchy, one pull request at a time.</p>
<p>A small, stateless Go app. No accounts, storage, secrets, or outbound requests.</p>
<p>Revision: <code>{{.SHA}}</code></p><p>Built: <code>{{.Built}}</code></p>
{{if .Deployed}}<p>{{.Deployed}}</p>{{end}}
<nav><a href="/healthz">Health</a> · <a href="/version">Version JSON</a></nav>
<footer>Deployed by patchy · built {{.Built}}</footer>
</main></body></html>
`))

func relativeTime(since time.Duration) string {
	minutes := int(since / time.Minute)
	switch {
	case minutes < 1:
		return "just now"
	case minutes == 1:
		return "1 minute ago"
	default:
		return fmt.Sprintf("%d minutes ago", minutes)
	}
}

func newHandler(sha, built string) http.Handler {
	info := version{SHA: sha, Built: built}
	builtAt, builtErr := time.Parse(time.RFC3339, built)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		data := pageData{SHA: sha, Built: built}
		if builtErr == nil {
			data.Deployed = "Deployed " + relativeTime(time.Since(builtAt))
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}
