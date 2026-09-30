// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

// netprobe is a one-shot isolation test, never an HTTP service.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"
)

type target struct {
	Name string
	Addr string
}

// These are the live cluster service IPs read before this throwaway PR. A
// changed service IP makes the result inconclusive and requires a new probe.
var deniedTargets = []target{
	{"instance-metadata", "169.254.169.254:80"},
	{"pod-identity-agent", "169.254.170.23:80"},
	{"kubernetes-api", "172.20.0.1:443"},
	{"patchy-egress-broker", "172.20.133.238:8080"},
	{"patchy-integration-controller", "172.20.53.113:8080"},
	{"patchy-source-controller", "172.20.80.122:9790"},
	{"patchy-status-server", "172.20.99.29:8080"},
	{"internet", "1.1.1.1:443"},
}

type result struct {
	Name    string `json:"name"`
	Target  string `json:"target"`
	Outcome string `json:"outcome"`
	Detail  string `json:"detail,omitempty"`
}

func denialOutcome(err error) string {
	if err == nil {
		return "CONNECTED"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "blocked"
	}
	if errors.Is(err, syscall.ENETUNREACH) || errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
		return "blocked"
	}
	// A refused connection proves only that nothing listened, not isolation.
	return "inconclusive"
}

func emit(r result) {
	if err := json.NewEncoder(os.Stdout).Encode(r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	const timeout = 3 * time.Second
	// Let the node agent attach policy before making any network connection.
	emit(result{Name: "startup-delay", Target: "pod", Outcome: "waiting", Detail: "60s"})
	time.Sleep(60 * time.Second)
	emit(result{Name: "startup-delay", Target: "pod", Outcome: "complete", Detail: "60s"})

	dialer := &net.Dialer{Timeout: timeout}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	code, tokenErr := imdsTokenPUT(ctx, dialer)
	cancel()
	if tokenErr != nil {
		emit(result{Name: "imds-v2-token-put", Target: "169.254.169.254:80", Outcome: "no-status", Detail: tokenErr.Error()})
	} else {
		emit(result{Name: "imds-v2-token-put", Target: "169.254.169.254:80", Outcome: "http-status", Detail: fmt.Sprint(code)})
		if code == 200 {
			os.Exit(1)
		}
	}

	resolver := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, "172.20.0.10:53")
	}}
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	addrs, err := resolver.LookupIPAddr(ctx, "kubernetes.default.svc.cluster.local.")
	cancel()
	failed := err != nil || len(addrs) == 0
	dns := result{Name: "cluster-dns", Target: "172.20.0.10:53", Outcome: "resolved"}
	if failed {
		dns.Outcome = "failed"
		if err != nil {
			dns.Detail = err.Error()
		}
	} else {
		dns.Detail = addrs[0].IP.String()
	}
	emit(dns)

	for _, t := range deniedTargets {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		conn, err := dialer.DialContext(ctx, "tcp", t.Addr)
		cancel()
		if conn != nil {
			_ = conn.Close()
		}
		outcome := denialOutcome(err)
		r := result{Name: t.Name, Target: t.Addr, Outcome: outcome}
		if err != nil {
			r.Detail = err.Error()
		}
		emit(r)
		if outcome != "blocked" {
			// Stop on the first reachable forbidden endpoint.
			os.Exit(1)
		}
	}
	if failed {
		os.Exit(1)
	}
}
