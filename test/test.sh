#!/usr/bin/env sh

docker build ./runner -t passage-runner:latest
docker run --network="passage" passage-runner:latest