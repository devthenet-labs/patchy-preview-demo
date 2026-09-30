// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"syscall"
	"testing"
)

func TestDenialOutcome(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"connected", nil, "CONNECTED"},
		{"timeout", os.ErrDeadlineExceeded, "blocked"},
		{"no-route", syscall.ENETUNREACH, "blocked"},
		{"permission", syscall.EPERM, "blocked"},
		{"refused-is-not-proof", syscall.ECONNREFUSED, "inconclusive"},
		{"wrapped-refusal", fmt.Errorf("dial: %w", syscall.ECONNREFUSED), "inconclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := denialOutcome(tc.err); got != tc.want {
				t.Fatalf("denialOutcome(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}
