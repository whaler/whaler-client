#!/bin/sh

GOOS=${GOOS:=linux}
GOARCH=${GOARCH:=amd64}

DIR="build/$GOOS/$GOARCH"
mkdir -p "$DIR"

FILE="whaler"
if [ "windows" = "$GOOS" ]; then
    FILE="whaler.exe"
fi

go get github.com/fatih/color
go get github.com/nareix/curl
go get github.com/fatih/flags
go get golang.org/x/crypto/ssh/terminal
go get github.com/inconshreveable/go-update
go build -o "$DIR/$FILE" whaler.go

md5sum --tag "$DIR/$FILE" > "$DIR/md5"
