#!/bin/bash

go mod tidy
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
go get -d golang.org/x/mobile/cmd/gomobile

build_apple() {
    rm -fr *.xcframework
    gomobile bind -target ios,iossimulator,macos -iosversion 15.0
}

build_android() {
    rm -fr *.jar
    rm -fr *.aar
    gomobile bind -target android -androidapi 28
}

download_geo() {
    go run main/main.go
}

echo "will build libxray for $1"
download_geo
if [ "$1" != "apple" ]; then
build_android
else
build_apple
fi
echo "build libxray done"
