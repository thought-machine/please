FROM golang:1.24-alpine
LABEL org.opencontainers.image.authors="please <at> thoughtmachine <dot> net"
LABEL org.opencontainers.image.source https://github.com/thought-machine/please

RUN apk add --no-cache git patch gcc g++ libc-dev bash libgcc xz protoc protobuf-dev perl-utils

# Ensure this is where we expect on the PATH
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go

WORKDIR /tmp
