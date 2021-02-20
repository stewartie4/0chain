#!/bin/sh
set -e

# TODO: don't forget to change back to normal script

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

docker run zchain_unit_test sh -c 'cd 0chain.net/smartcontract/; go test -tags bn256 0chain.net/smartcontract/...'
