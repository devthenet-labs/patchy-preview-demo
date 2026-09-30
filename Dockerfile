# Copyright 2026 DevTheNet Labs.
# SPDX-License-Identifier: MIT
FROM golang:1.26.6@sha256:0d1d3a794be25f809dd2cb3160d8c73276c4056a9f8242a138e908ddeee7b6b6 AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
COPY cmd/netprobe ./cmd/netprobe
ARG BUILD_SHA=dev
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath -buildvcs=false \
    -o /out/probe ./cmd/netprobe

FROM scratch
COPY --from=build /out/probe /probe
USER 65532:65532
ENTRYPOINT ["/probe"]
