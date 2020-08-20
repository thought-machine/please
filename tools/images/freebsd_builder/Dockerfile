FROM ubuntu:focal
MAINTAINER peter.ebden@gmail.com

# A few miscellaneous dependencies.
RUN apt-get update && apt-get install -y curl git locales && apt-get clean

# Go
RUN curl -fsSL https://dl.google.com/go/go1.15.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt
ENV GOOS=freebsd
ENV GOARCH=amd64
RUN go install std

# Locale
RUN locale-gen en_GB.UTF-8

WORKDIR /tmp
