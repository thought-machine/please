FROM ubuntu:bionic
MAINTAINER peter.ebden@gmail.com

# Most dependencies; Python, Java, Clang and Ruby (only used for fpm)
RUN apt-get update && \
    apt-get install -y python3 python3-dev python3-pip openjdk-11-jdk-headless \
    curl unzip git locales pkg-config zlib1g-dev \
    clang libunittest++-dev ruby ruby-dev rubygems golint && \
    apt-get clean

# Go - we are specifically testing compatibility with 1.9 here.
RUN curl -fsSL https://dl.google.com/go/go1.9.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Locale
RUN locale-gen en_GB.UTF-8

# fpm
RUN gem install --no-ri --no-rdoc fpm

# Welcome message
COPY /motd.txt /etc/motd
RUN echo 'cat /etc/motd' >> /etc/bash.bashrc
WORKDIR /tmp
