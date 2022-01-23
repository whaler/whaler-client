#!/bin/bash

set -e

SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

GOOS=linux GOARCH=amd64 "${SCRIPT_DIR}/build.sh" $@
GOOS=linux GOARCH=arm64 "${SCRIPT_DIR}/build.sh" $@
GOOS=linux GOARCH=arm GOARM=6 "${SCRIPT_DIR}/build.sh" $@
GOOS=linux GOARCH=arm GOARM=7 "${SCRIPT_DIR}/build.sh" $@

GOOS=darwin GOARCH=amd64 "${SCRIPT_DIR}/build.sh" $@
GOOS=darwin GOARCH=arm64 "${SCRIPT_DIR}/build.sh" $@

GOOS=windows GOARCH=amd64 "${SCRIPT_DIR}/build.sh" $@
