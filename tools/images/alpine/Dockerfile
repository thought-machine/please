FROM golang:1.15-alpine
MAINTAINER peter.ebden@gmail.com

RUN apk update && apk add --no-cache git patch gcc g++ libc-dev bash libgcc

# Ensure this is where we expect on the PATH
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go

# Configure builds to be static
COPY /plzconfig /etc/please/plzconfig
WORKDIR /tmp
