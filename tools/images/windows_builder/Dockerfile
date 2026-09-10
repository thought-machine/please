FROM ubuntu:noble
LABEL org.opencontainers.image.authors="please <at> thoughtmachine <dot> net"
LABEL org.opencontainers.image.source=https://github.com/thought-machine/please

# A few miscellaneous dependencies. MinGW is here for the C/C++ rules, which cross-compile to
# Windows with it; the Go cross-build itself needs no C toolchain.
RUN apt-get update && apt-get install -y curl git gcc xz-utils g++-mingw-w64-x86-64 && apt-get clean

# Go
RUN curl -fsSL https://dl.google.com/go/go1.26.1.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt
RUN GOOS=windows go install std

WORKDIR /tmp
