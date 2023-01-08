#!/bin/bash

set -e

mkdir -p tmp

go version | { read _ _ v _; echo -n ${v#go}; } > tmp/go_version.txt
git describe --tags --dirty | tr -d '\n' > tmp/version.txt
git rev-parse --short HEAD | tr -d '\n' > tmp/commit_hash.txt
