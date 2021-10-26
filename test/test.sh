#!/usr/bin/env sh

docker build ./runner -t passage-runner:latest
docker run  \
  --name="passage-test" --rm \
  --network="passage" \
  --volume test_ssh-public-keys:/public-keys \
  passage-runner:latest