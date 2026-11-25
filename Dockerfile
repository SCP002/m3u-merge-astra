# syntax=docker/dockerfile:1

FROM golang:1.27.3-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Version is taken from build.sh unless set explicitly.
ARG VERSION
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    version="${VERSION:-$(sed -n 's/^version="\(.*\)"$/\1/p' build.sh)}" \
    && CGO_ENABLED=0 go build -trimpath \
        -ldflags "-s -w -X m3u-merge-astra/version.Version=${version}" \
        -o /out/m3u-merge-astra .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# https://astra.cesbo.com/latest requires an active subscription.
# For a lifetime license pin a build from https://astra.cesbo.com/en/quick-start/versions/,
# e.g. https://cdn.cesbo.com/astra/builds/astra-250221
ARG ASTRA_URL=https://astra.cesbo.com/latest
ADD --chmod=755 ${ASTRA_URL} /usr/bin/astra
RUN astra -v

COPY --from=build /out/m3u-merge-astra /usr/bin/m3u-merge-astra
RUN m3u-merge-astra --version

# Working directory of m3u-merge-astra: its config (m3u-merge-astra.yaml) and local M3U files.
WORKDIR /m3u-merge-astra

ENTRYPOINT ["/usr/bin/astra"]
CMD ["-c", "/etc/astra/astra.conf", "-p", "8000"]
