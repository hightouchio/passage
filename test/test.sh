#!/usr/bin/env bash

docker network ls
docker build ./test/runner -t passage-runner:latest

docker run  \
  --name="passage-test-reverse" --rm \
  --network="passage" \
  --env EXPECTED_SERVICE_RESPONSE="You're talking to the remote service!" \
  --volume test_reverse-tunnel-config:/reverse_tunnel \
  passage-runner:latest /test-reverse.rb

docker run  \
  --name="passage-test-standard" --rm \
  --network="passage" \
  --env EXPECTED_SERVICE_RESPONSE="You're talking to the remote service!" \
  --volume test_bastion-ssh-config:/bastion_ssh \
  passage-runner:latest /test-standard.rb