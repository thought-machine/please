FROM ubuntu:xenial
MAINTAINER peter.ebden@gmail.com

# Basic dependencies, Python and Java
RUN apt-get update && \
    apt-get install -y python2.7 python3.5 python3-pip libpython2.7-dev openjdk-8-jdk-headless \
    curl unzip git locales pkg-config zlib1g-dev \
    clang ruby ruby-dev rubygems
RUN pip3 install cffi

# Go
RUN curl -fsSL https://storage.googleapis.com/golang/go1.9.1.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# C++
# Ubuntu's packaged version of UnitTest++ is older than we'd like and has some
# idiosyncratic naming (unittest++ vs. UnitTest++ etc).
# TODO(peterebden): Update this to 17.10 which fixes all the above.
RUN curl -fsSL https://github.com/unittest-cpp/unittest-cpp/releases/download/v2.0.0/unittest-cpp-2.0.0.tar.gz | tar -xzC /tmp && \
    cd /tmp/unittest-cpp-2.0.0 && ./configure --prefix=/usr && make -j4 && make install && \
    rm -r /tmp/unittest-cpp-2.0.0

# Protocol buffers
RUN curl -fsSLo /tmp/protoc.zip https://github.com/google/protobuf/releases/download/v3.4.0/protoc-3.4.0-linux-x86_64.zip && \
    unzip /tmp/protoc.zip -x readme.txt -d /usr && \
    rm /tmp/protoc.zip

# Locale
RUN locale-gen en_GB.UTF-8

# fpm
RUN gem install --no-ri --no-rdoc fpm

# Welcome message
COPY /motd.txt /etc/motd
RUN echo 'cat /etc/motd' >> /etc/bash.bashrc

# Linters etc
WORKDIR /tmp
RUN go get github.com/golang/lint/golint && mv ~/go/bin/golint /usr/local/bin && rm -rf ~/go
