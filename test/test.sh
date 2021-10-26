#!/usr/bin/env sh

docker build ./runner -t passage-runner:latest
docker run  \
  --name="passage-test" --rm \
  --network="passage" \
  --env EXPECTED_SERVICE_RESPONSE="You're talking to the remote service!" \
  --volume test_bastion-ssh-config:/bastion-ssh \
  passage-runner:latest