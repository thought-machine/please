FROM golang:1.26-alpine
LABEL org.opencontainers.image.authors="please <at> thoughtmachine <dot> net"
LABEL org.opencontainers.image.source=https://github.com/thought-machine/please

RUN apk add --no-cache git patch gcc g++ libc-dev bash libgcc xz protoc protobuf-dev perl-utils shadow-uidmap curl

# Subordinate id delegation for sandbox tests
RUN printf 'root:0:65536\n' >> /etc/subuid && \
    printf 'root:0:65536\n' >> /etc/subgid

# Ensure this is where we expect on the PATH
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go

WORKDIR /tmp
