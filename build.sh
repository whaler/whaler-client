#!/bin/sh

GOOS=${GOOS:=linux}
GOARCH=${GOARCH:=amd64}

DIR="$GOOS/$GOARCH"
mkdir -p "$DIR"

FILE="whaler"
if [ "windows" = "$GOOS" ]; then
    FILE="whaler.exe"
fi

go get github.com/fatih/color
go get github.com/fatih/flags
go get golang.org/x/crypto/ssh/terminal
go build -o "$DIR/$FILE" whaler.go
