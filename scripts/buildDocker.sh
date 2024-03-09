#!/usr/bin/env sh

set -e

source "$(dirname $0)/_variables.sh"

docker run --rm -v ${PWD}:/opt -it golang:1.22-alpine3.19 sh -c \
  "cd /opt && go build -o ${BINARY}"
