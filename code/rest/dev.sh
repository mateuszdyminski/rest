#!/bin/bash

usage() {
	cat <<EOF
Usage: $(basename $0) <command>

Wrappers around core binaries:
    build                  Builds the rest.
    buildPush              Builds docker image and pushes it to DockerHub.
EOF
	exit 1
}

CMD="$1"
shift
case "$CMD" in
	build)
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main -a -tags netgo .
	;;
	buildPush)
		docker build -t mateuszdyminski/rest:latest . && docker push mateuszdyminski/rest
	;;
	*)
		usage
	;;
esac