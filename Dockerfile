# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

# Build Geth in a stock Go builder container
FROM golang:1.26.8-bookworm AS builder

# Get dependencies - will also be cached if we won't change go.mod/go.sum
COPY go.mod go.sum /go-ethereum/
RUN cd /go-ethereum && go mod download

ADD . /go-ethereum

RUN cd /go-ethereum && env GO111MODULE=on go run build/ci.go install ./cmd/geth

# Pull Geth into a second stage deploy container
FROM ubuntu:24.04

RUN apt-get -qq update \
    && DEBIAN_FRONTEND=noninteractive apt-get -qq upgrade -y \
    && apt-get -qq install -y --no-install-recommends ca-certificates netcat-openbsd curl \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["geth"]

# Add some metadata labels to help programatic image consumption
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
