// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadStatusCodeDoesNotReadTokenBody(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	for _, line := range []string{"oops\n", "HTTP/1.1 nope\n", strings.Repeat("x", 129)} {
		if _, err := readStatusCode(strings.NewReader(line)); err == nil {
			t.Fatalf("accepted malformed status %q", line)
		}
	}
}
