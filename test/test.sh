#!/usr/bin/env sh

set -e

# Teardown environment.
cleanup() {
  docker-compose -f test/docker-compose.yml down --volumes
  exit $?
}
trap cleanup EXIT

# Spin up environment.
docker-compose -f test/docker-compose.yml build passage
docker-compose -f test/docker-compose.yml up -d

# Execute tests.
./test/runner.sh