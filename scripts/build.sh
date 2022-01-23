#!/bin/bash

set -e

SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
WORKDIR="$( dirname "${SCRIPT_DIR}" )"

cd "${WORKDIR}"

GOOS=${GOOS:=linux}
GOARCH=${GOARCH:=amd64}
BUILD_DIR=${BUILD_DIR:=.build}
VERSION=${VERSION:=0.x}
ARCHIVE=NO

ARCH="${GOARCH}"
if [ "arm" = "${ARCH}" ]; then
    ARCH="arm${GOARM}"
fi

for i in "$@"; do
case $i in
    --archive)
        ARCHIVE=YES
        shift
    ;;
    *)
        # unknown option
    ;;
esac
done

DIR="${BUILD_DIR}/${GOOS}/${ARCH}"
mkdir -p "${DIR}"

FILE="whaler"
if [ "windows" = "${GOOS}" ]; then
    FILE="${FILE}.exe"
fi

SHA=$( git rev-parse HEAD 2>/dev/null | head -c7 )
if [ -z "${SHA}" ]; then
    SHA="dev"
fi

PACKAGE="cmd/whaler/main.go"
LDFLAGS="-s -X main.VERSION=${VERSION}-${SHA}"

if [ "linux" = "${GOOS}" ]; then
    CGO_ENABLED=0 go build -a -installsuffix ${ARCH} -o "${DIR}/${FILE}" -ldflags "${LDFLAGS}" ${PACKAGE}
else
    go build -o "${DIR}/${FILE}" -ldflags "${LDFLAGS}" ${PACKAGE}
fi

md5_sum() {
    local IN_FILE=${1}
    local OUT_FILE=${2}

    if [[ "$OSTYPE" == "darwin"* ]]; then
        md5 "${IN_FILE}" > "${OUT_FILE}"
    else
        md5sum --tag "${IN_FILE}" > "${OUT_FILE}"
    fi
}

md5_sum "${DIR}/${FILE}" "${DIR}/md5"

if [ "YES" = "${ARCHIVE}" ]; then
    cd "${DIR}"
    cp "${WORKDIR}/LICENSE" ./
    cp "${WORKDIR}/README.md" ./
    TAR_FILE="${WORKDIR}/${BUILD_DIR}/whaler_${GOOS}_${ARCH}.tar.gz"
    tar -czf "${TAR_FILE}" *
    md5_sum "${TAR_FILE}" "${TAR_FILE}.md5"
    rm -rf "${WORKDIR}/$( dirname "${DIR}" )"
fi

echo "${VERSION}-${SHA}" > "${WORKDIR}/${BUILD_DIR}/version"