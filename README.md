# libXray

[简体中文](./readme/README.zh_CN.md)

This is a wrapper around [Xray-core](https://github.com/XTLS/Xray-core) to improve the client development experience.

# Note

1. This repository has few maintainers. If you do not report a bug or initiate a PR, your issue will be ignored.
2. This repository does not guarantee API stability, you need to adapt it yourself.
3. This repository is only compatible with the latest release of Xray-core.

# Versioning

Releases use CalVer in the form `v<YY>.<M>.<D>` (e.g. `v26.3.27` = 2026-03-27).
Because Go modules require any module with major version `>= 2` to encode the
major in its import path, every CalVer release is mirrored onto a Go-friendly
SemVer tag on the same commit:

| CalVer tag | Go-import tag |
| ---------- | ------------- |
| `v26.3.27` | `v1.260327.0` |

Go consumers should pin against the SemVer mirror:

```shell
go get github.com/xtls/libxray@v1.260327.0
```

The mirror tag is created automatically by
[`.github/workflows/release-go-mirror.yml`](./.github/workflows/release-go-mirror.yml)
on every CalVer push. Existing CalVer tags can be backfilled with
[`scripts/backfill-semver-tags.sh`](./scripts/backfill-semver-tags.sh).

# Features

## build

Compile script. It is recommended to always use this script to compile libXray. We will not answer questions caused by using other compilation methods.

depends on git and go.

By default, the build script does not clone [Xray-core](https://github.com/XTLS/Xray-core). It uses Go modules and pins Xray-core to main commit `d5bc58dc` through its pseudo-version.
Pass the optional `local` argument to use an existing local checkout at `../Xray-core` through a Go module `replace`.

### Usage

```shell
# Android (min Android API level is 21)
python3 build/main.py android
python3 build/main.py android local

# Apple (gomobile or go)
python3 build/main.py apple gomobile
python3 build/main.py apple go
python3 build/main.py apple gomobile local
python3 build/main.py apple go local

# Linux
python3 build/main.py linux
python3 build/main.py linux local

# Windows
python3 build/main.py windows
python3 build/main.py windows local

```

### Android

use [gomobile](https://github.com/golang/mobile) .

### iOS && macOS

#### 1. use gomobile

Need "iOS Simulator Runtime".

This is the best choice for general scenarios and will not conflict with other frameworks.

Supports iOS, iOSSimulator, macOS, macCatalyst.

But it is not possible to set the minimum macOS version, which will cause some warnings when compiling. And it does not support tvOS.

#### 2. use cgo

Need "iOS Simulator Runtime" and "tvOS Simulator Runtime".

Support more compilation options, output c header files.

This works well when you use ffi for integration. For example, integration with swift, kotlin, dart.

Support iOS, iOSSimulator, macOS, tvOS.

The product `LibXray.xcframework` contains **module.modulemap**. When using
Swift, import it as module `LibXray`.

### Linux

depend on gcc and g++.

### Windows

depend on LLVM MinGW.

you can use winget to install [LLVM MinGW](https://github.com/mstorsjo/llvm-mingw).

```shell
winget install MartinStorsjo.LLVM-MinGW.UCRT
```

## API

libXray exposes a single structured entrypoint:

```go
func Invoke(requestJSON string) string
```

The C export is:

```c
char* CGoInvoke(char* requestJSON);
```

The request is a JSON object:

```json
{
  "apiVersion": 1,
  "method": "runXray",
  "payload": {
    "configPath": "/path/to/config.json"
  }
}
```

The response is a JSON object:

```json
{
  "success": true,
  "data": {},
  "error": ""
}
```

Design notes:

1. A top-level `env` field is ignored and has no effect. Xray-core runtime
   environment options belong in the root `env` object of the Xray config.
2. `SetTunFd` has been removed. When the fd is only known at runtime, write
   `xray.tun.fd` into the Xray config root `env` object before calling
   `runXray`.
3. `countGeoData` is not backed by an Xray config, so its `datDir` is passed in
   the method payload.
4. The complete UTF-8 encoded Invoke request and response JSON envelopes are
   limited to 16 MiB. If either limit is exceeded, Invoke returns a failure
   response with `success: false`, `data: null`, and a size-limit error.

Supported methods:

```text
getFreePorts
convertShareLinksToXrayJson
convertXrayJsonToShareLinks
countGeoData
ping
testXray
runXray
runXrayFromJson
stopXray
xrayVersion
getXrayState
```

## controller

Used to solve the socket protect problem on Android.

## geo

### count

Read geo files and count the categories and rules.

## main

Download geosite.dat and geoip.dat and count them.

## memory

Only executed on iOS, GC is initiated once a second. This can alleviate memory pressure on iOS.

## nodep

### file

Write data to a file.

### measure

Speed ​​test the Xray configuration.

### port

Get free ports.

## share

libXray uses `sendThrough` to store outbound names.

### clash_meta

Parse Clash.Meta configuration.

### generate_share

convert Xray Json to VMessAEAD/VLESS sharing protocol.

### parse_share

convert VMessAEAD/VLESS sharing protocol to Xray Json.

convert VMessQRCode to Xray Json.

### vmess

convert VMessQRCode to Xray Json.

### xray_json

Some tools used to parse shared links.

## xray

### ping

Latency testing.

### metrics

Refer to the following configuration:

```json
{
  "metrics" : {
    "listen": "127.0.0.1:49227"
  },
  "policy" : {
    "system" : {
      "statsInboundDownlink" : true,
      "statsInboundUplink" : true,
      "statsOutboundDownlink" : true,
      "statsOutboundUplink" : true
    }
  },
  "stats" : {}
}
```

The metrics server exposes the Xray runtime counters through HTTP. For example,
when `listen` is `127.0.0.1:49227`, read:

```text
http://localhost:49227/debug/vars
```

Note:

1. When testing latency or validating configuration, make sure `metrics` is `null`.

2. Metrics only needs the `listen` field in this wrapper. Query `/debug/vars` directly with an HTTP client instead of going through libXray.

### validation

Verify the Xray configuration.

### xray

Start and stop Xray instances.

# Credits

[Project X](https://github.com/XTLS/Xray-core)

[VMessPing](https://github.com/v2fly/vmessping)

[FreePort](https://github.com/phayes/freeport)

# License

This repository is based on the MIT License.
