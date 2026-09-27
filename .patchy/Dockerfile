# Copyright 2026 DevTheNet Labs.
# SPDX-License-Identifier: MIT
# Toolchain only: application source is never baked into an agent image.
FROM golang:1.26.6@sha256:0d1d3a794be25f809dd2cb3160d8c73276c4056a9f8242a138e908ddeee7b6b6 AS toolchain
FROM ghcr.io/devthenet-labs/patchy/agent-base:v0.12.7@sha256:8a000e76bcfde68791e2b9ee6ae38ad93ccdae40bc204d7134ae932ccde301e8
USER 0:0
COPY --from=toolchain /usr/local/go /usr/local/go
RUN mkdir -p /usr/local/bin \
    && ln -s /usr/local/go/bin/go /usr/local/bin/go \
    && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt \
    && mkdir -p /opt/go/pkg/mod
ENV GOTOOLCHAIN=local GOMODCACHE=/opt/go/pkg/mod GOPROXY=off CGO_ENABLED=0
WORKDIR /tmp/deps
COPY go.mod ./
RUN go mod download
LABEL org.opencontainers.image.source="https://github.com/devthenet-labs/patchy-preview-demo" \
      org.opencontainers.image.title="patchy-preview-demo agent toolchain"
WORKDIR /workspace
USER 65532:65532
