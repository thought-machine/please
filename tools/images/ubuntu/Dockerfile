FROM ubuntu:jammy
MAINTAINER peter.ebden@gmail.com


ENV DEBIAN_FRONTEND noninteractive
ENV DEBCONF_NONINTERACTIVE_SEEN true

# Python and other dependencies.
RUN truncate -s0 /tmp/preseed.cfg; \
    echo "tzdata tzdata/Areas select Europe" >> /tmp/preseed.cfg; \
    echo "tzdata tzdata/Zones/Europe select London" >> /tmp/preseed.cfg; \
    apt-get update && \
    apt-get install -y python3 python3-dev python3-pip time \
    curl unzip git locales pkg-config zlib1g-dev psmisc awscli && \
    apt-get clean

# Go - we want a specific package version here.
RUN curl -fsSL https://dl.google.com/go/go1.22.0.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Locale
RUN locale-gen en_GB.UTF-8

# Welcome message
COPY /motd.txt /etc/motd
RUN echo 'cat /etc/motd' >> /etc/bash.bashrc
WORKDIR /tmp
