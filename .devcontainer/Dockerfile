# Pin the Debian base flavor for reproducibility; bump GO_VERSION via build arg.
ARG GO_VERSION=1.25
FROM golang:${GO_VERSION}-bookworm

WORKDIR /workspaces/app
ENV GOPATH=/go
