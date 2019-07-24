FROM ubuntu:bionic
MAINTAINER peter.ebden@gmail.com

RUN apt-get update && apt-get install -y nginx-extras && apt-get clean
COPY /webdav.conf /etc/nginx/conf.d/webdav.conf
CMD [ "nginx", "-g", "daemon off;" ]
EXPOSE 8080/tcp
