#!/bin/bash

PROJECT_ROOT=$(dirname "${BASH_SOURCE}")/..

# Register function to be called on EXIT to remove generated binary.
function cleanup {
  rm "${PROJECT_ROOT}/artifacts/simple-image/namespace-reservation-server"
}
trap cleanup EXIT

pushd "${PROJECT_ROOT}"
cp -v _output/bin/namespace-reservation-server ./artifacts/simple-image/namespace-reservation-server
docker build -t namespace-reservation-server:latest ./artifacts/simple-image
popd