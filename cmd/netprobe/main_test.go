// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestClassify(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"connected", nil, "REACHABLE"},
		{"timeout", os.ErrDeadlineExceeded, "blocked"},
		{"no-route", syscall.ENETUNREACH, "blocked"},
		{"permission", syscall.EPERM, "blocked"},
		{"refused-is-not-proof", syscall.ECONNREFUSED, "inconclusive"},
		{"wrapped-refusal", fmt.Errorf("dial: %w", syscall.ECONNREFUSED), "inconclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := classify(tc.err); got != tc.want {
				t.Fatalf("classify(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestReadStatusCodeDoesNotReadTokenBody(t *testing.T) {
	const statusLine = "HTTP/1.1 200 OK\r\n"
	const rest = "Content-Length: 12\r\n\r\nsecret-token"
	r := bytes.NewReader([]byte(statusLine + rest))
	code, err := readStatusCode(r)
	if err != nil || code != 200 {
		t.Fatalf("readStatusCode = %d, %v", code, err)
	}
	if r.Len() != len(rest) {
		t.Fatalf("read past status line: %d bytes left, want %d", r.Len(), len(rest))
	}
}

func TestReadStatusCodeRejectsMalformed(t *testing.T) {
	for _, line := range []string{"oops\n", "HTTP/1.1 nope\n", strings.Repeat("x", 129)} {
		if _, err := readStatusCode(strings.NewReader(line)); err == nil {
			t.Fatalf("accepted malformed status %q", line)
		}
	}
}
