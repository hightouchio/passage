#!/usr/bin/env sh

# Spin up environment.
docker-compose -f test/docker-compose.yml build passage
docker-compose -f test/docker-compose.yml up -d

# Execute tests.
./test/runner.sh

# Teardown environment.
docker-compose -f test/docker-compose.yml down --volumes