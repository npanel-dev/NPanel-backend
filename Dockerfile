# syntax=docker/dockerfile:1
# Multi-arch build: TARGETOS/TARGETARCH/TARGETVARIANT are auto-injected by buildx
# when invoked with `docker buildx build --platform linux/amd64,linux/arm64`.

# OCI annotations (overridable via build args)
ARG SOURCE=https://github.com/npanel-dev/NPanel-backend
ARG TITLE=NPanel Backend
ARG VERSION=dev
ARG REVISION=

FROM golang:1.26-bookworm AS builder

ARG TARGETOS=linux
ARG TARGETARCH
ARG TARGETVARIANT
ARG VERSION=dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build -trimpath -ldflags "-s -w -X main.Version=${VERSION}" \
	-o /out/npanel ./cmd/npanel

FROM debian:stable-slim

ARG SOURCE
ARG TITLE
ARG VERSION
ARG REVISION

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates \
		netbase \
		&& rm -rf /var/lib/apt/lists/ \
		&& apt-get autoremove -y \
		&& apt-get autoclean -y

COPY --from=builder /out/npanel /app/npanel

WORKDIR /app

EXPOSE 8081 9012
VOLUME ["/data/conf"]

# OCI standard labels (https://github.com/opencontainers/image-spec/blob/main/annotations.md)
LABEL org.opencontainers.image.source="${SOURCE}" \
	  org.opencontainers.image.title="${TITLE}" \
	  org.opencontainers.image.version="${VERSION}" \
	  org.opencontainers.image.revision="${REVISION}"

CMD ["/app/npanel", "-conf", "/data/conf"]
