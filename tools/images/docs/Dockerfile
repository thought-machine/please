FROM ubuntu:bionic
MAINTAINER peter.ebden@gmail.com

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y openssh-client ca-certificates python3 awscli
COPY sftp.txt /root/sftp.txt
