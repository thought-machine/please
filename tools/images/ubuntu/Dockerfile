FROM ubuntu:bionic
MAINTAINER peter.ebden@gmail.com

# Most dependencies; Go, Python, Java, Clang and Ruby (only used for fpm)
RUN apt-get update && \
    apt-get install -y python3 python3-dev python3-pip openjdk-8-jdk-headless \
    curl git locales pkg-config zlib1g-dev \
    clang libunittest++-dev ruby ruby-dev rubygems golang-go golint && \
    apt-get clean

# Locale
RUN locale-gen en_GB.UTF-8

# fpm
RUN gem install --no-ri --no-rdoc fpm

# Welcome message
COPY /motd.txt /etc/motd
RUN echo 'cat /etc/motd' >> /etc/bash.bashrc
WORKDIR /tmp
