FROM golang:1.9.2-alpine

RUN mkdir -p /usr/share/rest

ADD main /usr/share/rest/main

EXPOSE 8080

ENTRYPOINT /usr/share/rest/main
