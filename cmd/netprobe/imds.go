// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
)

// imdsTokenPUT reads only the HTTP status line. It never reads a token,
// headers, a response body, or credentials. connected distinguishes an
// established TCP connection from a network-policy denial.
func imdsTokenPUT(ctx context.Context, dialer *net.Dialer, addr string) (status int, connected bool, err error) {
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, false, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return 0, true, err
		}
	}
	_, err = io.WriteString(conn, "PUT /latest/api/token HTTP/1.1\r\nHost: 169.254.169.254\r\nX-aws-ec2-metadata-token-ttl-seconds: 60\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	if err != nil {
		return 0, true, err
	}
	status, err = readStatusCode(conn)
	return status, true, err
}

func readStatusCode(r io.Reader) (int, error) {
	var line []byte
	for len(line) < 128 {
		var b [1]byte
		if _, err := r.Read(b[:]); err != nil {
			return 0, err
		}
		if b[0] == '\n' {
			fields := strings.Fields(string(line))
			if len(fields) < 2 || !strings.HasPrefix(fields[0], "HTTP/") {
				return 0, errors.New("invalid HTTP status line")
			}
			return strconv.Atoi(fields[1])
		}
		line = append(line, b[0])
	}
	return 0, errors.New("HTTP status line too long")
}
