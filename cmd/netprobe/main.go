// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

// netprobe is a disposable, one-shot network-isolation test. It has no listener.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

type target struct {
	name string
	addr string
	imds bool
}

// Pinned Service IPs are checked against the cluster before using the image.
// A changed IP makes the probe inconclusive; none is resolved through DNS.
var forbidden = []target{
	{name: "imds-v2-token-put", addr: "169.254.169.254:80", imds: true},
	{name: "pod-identity-agent", addr: "169.254.170.23:80"},
	{name: "kubernetes-api", addr: "172.20.0.1:443"},
	{name: "patchy-egress-broker", addr: "172.20.133.238:8080"},
	{name: "patchy-integration-controller", addr: "172.20.53.113:8080"},
	{name: "patchy-source-controller", addr: "172.20.80.122:9790"},
	{name: "patchy-status-server", addr: "172.20.99.29:8080"},
	{name: "internet", addr: "1.1.1.1:443"},
}

type result struct {
	Name      string `json:"name"`
	Worker    int    `json:"worker,omitempty"`
	Attempt   int    `json:"attempt,omitempty"`
	ElapsedMS int64  `json:"elapsed_ms"`
	Outcome   string `json:"outcome"`
	Status    int    `json:"status,omitempty"`
}

func classify(err error) string {
	if err == nil {
		return "REACHABLE"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "blocked"
	}
	if errors.Is(err, syscall.ENETUNREACH) || errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
		return "blocked"
	}
	// A connection refusal is not evidence of isolation: the listener might
	// simply be down. Other errors are likewise inconclusive.
	return "inconclusive"
}

func probe(ctx context.Context, dialer *net.Dialer, t target, worker, attempt int, start time.Time) result {
	ctx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	r := result{Name: t.name, Worker: worker, Attempt: attempt}
	if t.imds {
		status, connected, err := imdsTokenPUT(ctx, dialer, t.addr)
		r.Status = status
		if connected {
			r.Outcome = "REACHABLE"
		} else {
			r.Outcome = classify(err)
		}
	} else {
		conn, err := dialer.DialContext(ctx, "tcp", t.addr)
		if conn != nil {
			conn.Close()
		}
		r.Outcome = classify(err)
	}
	r.ElapsedMS = time.Since(start).Milliseconds()
	return r
}

func burst(start time.Time, emit func(result)) (reachable, inconclusive bool) {
	const workers = 4
	const attempts = 4
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dialer := &net.Dialer{}
	begin := make(chan struct{})
	results := make(chan result, len(forbidden)*workers*attempts)
	var wg sync.WaitGroup
	for _, t := range forbidden {
		for worker := 1; worker <= workers; worker++ {
			wg.Add(1)
			go func(t target, worker int) {
				defer wg.Done()
				<-begin
				for attempt := 1; attempt <= attempts; attempt++ {
					if ctx.Err() != nil {
						return
					}
					r := probe(ctx, dialer, t, worker, attempt, start)
					results <- r
					if r.Outcome == "REACHABLE" {
						return
					}
				}
			}(t, worker)
		}
	}
	// Releasing all workers is the first network-related operation in main.
	close(begin)
	go func() {
		wg.Wait()
		close(results)
	}()
	for r := range results {
		emit(r)
		if r.Outcome == "REACHABLE" {
			reachable = true
			cancel() // Stop all subsequent attempts immediately.
		} else if r.Outcome == "inconclusive" {
			inconclusive = true
		}
	}
	return reachable, inconclusive
}

func dnsOnce(ctx context.Context) error {
	_, err := net.DefaultResolver.LookupHost(ctx, "kubernetes.default.svc.cluster.local")
	return err
}

func emit(r result) {
	_ = json.NewEncoder(os.Stdout).Encode(r)
}

func main() {
	start := time.Now()
	reachable, inconclusive := burst(start, emit)
	if reachable {
		emit(result{Name: "summary", ElapsedMS: time.Since(start).Milliseconds(), Outcome: "SECURITY_FAILURE"})
		// Leave time for the observer to delete this Deployment before a restart.
		time.Sleep(60 * time.Second)
		os.Exit(2)
	}
	if inconclusive {
		emit(result{Name: "summary", ElapsedMS: time.Since(start).Milliseconds(), Outcome: "INCONCLUSIVE"})
		time.Sleep(60 * time.Second)
		os.Exit(3)
	}
	// DNS may become available after policy programming. No forbidden target
	// is retried after the initial burst.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := dnsOnce(ctx)
		cancel()
		if err == nil {
			emit(result{Name: "dns", ElapsedMS: time.Since(start).Milliseconds(), Outcome: "ok"})
			emit(result{Name: "summary", ElapsedMS: time.Since(start).Milliseconds(), Outcome: "PASS"})
			time.Sleep(30 * time.Second)
			return
		}
		time.Sleep(time.Second)
	}
	emit(result{Name: "summary", ElapsedMS: time.Since(start).Milliseconds(), Outcome: "DNS_FAILURE"})
	time.Sleep(60 * time.Second)
	os.Exit(4)
}
