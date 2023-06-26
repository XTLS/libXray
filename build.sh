#!/bin/bash

go mod tidy
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
go get -d golang.org/x/mobile/cmd/gomobile

build_apple() {
    rm -fr *.xcframework
    gomobile bind -target ios,iossimulator,macos
}

build_android() {
    rm -fr *.jar
    rm -fr *.aar
    gomobile bind -target android -androidapi 28
}

echo "will build libxray for $1"
if [ "$1" != "apple" ]; then
build_android
else
build_apple
fi
