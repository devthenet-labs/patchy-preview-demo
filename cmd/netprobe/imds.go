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

// imdsTokenPUT observes only the HTTP status line. It does not read headers,
// the response body (which would contain a token), or any credentials.
func imdsTokenPUT(ctx context.Context, dialer *net.Dialer) (int, error) {
	conn, err := dialer.DialContext(ctx, "tcp", "169.254.169.254:80")
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return 0, err
		}
	}
	_, err = io.WriteString(conn, "PUT /latest/api/token HTTP/1.1\r\nHost: 169.254.169.254\r\nX-aws-ec2-metadata-token-ttl-seconds: 60\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	if err != nil {
		return 0, err
	}
	return readStatusCode(conn)
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
