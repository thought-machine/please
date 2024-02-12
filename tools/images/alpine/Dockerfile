FROM golang:1.22-alpine
MAINTAINER peter.ebden@gmail.com

RUN apk update && apk add --no-cache git patch gcc g++ libc-dev bash libgcc xz protoc protobuf-dev perl-utils

# Ensure this is where we expect on the PATH
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go

WORKDIR /tmp
