# libXray

[简体中文](./readme/README.zh_CN.md)

This is a wrapper around [Xray-core](https://github.com/XTLS/Xray-core) to improve the client development experience.

# Note

1. This repository has few maintainers. If you do not report a bug or initiate a PR, your issue will be ignored.
2. This repository does not guarantee API stability, you need to adapt it yourself.
3. This repository is only compatible with the latest release of Xray-core.

# Features

## build

Compile script. It is recommended to always use this script to compile libXray. We will not answer questions caused by using other compilation methods.

### Usage

```shell
# Android (optional: specify Android API level, default is 21)
python3 build/main.py android [api-level]

# Apple (gomobile or go)
python3 build/main.py apple gomobile
python3 build/main.py apple go

# Linux
python3 build/main.py linux

# Windows
python3 build/main.py windows

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

Note: The product `LibXray.xcframework` does not contain **module.modulemap**. When using swift, you need to create a bridge file.

### Linux

depend on clang and clang++.

### Windows

depend on [LLVM MinGW](https://github.com/mstorsjo/llvm-mingw), you can install it using winget.

```shell
winget install MartinStorsjo.LLVM-MinGW.UCRT
```

## controller

Used to solve the socket protect problem on Android.

## dns

Used to solve server address resolution issues on Android, Linux, and Windows. If not handled, the DNS traffic will be resent to the tun device, resulting in failure to initiate a connection.

## geo

### count

Read geo files and count the categories and rules.

### read

Read the Xray Json configuration and extract the geo file name used.

### thin

Read the Xray Json configuration and cut the geo file used.

## main

Download geosite.dat and geoip.dat and count them.

## memory

Only executed on iOS, GC is initiated once a second. This can alleviate memory pressure on iOS.

## nodep

### file

Write data to a file.

### measure

Speed ​​test the Xray configuration.

### model

The response body of the wrapper interface.

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

### stats

Refer to the following configuration:

```json
{
  "metrics" : {
    "tag" : "metrics",
    "listen": "[::1]:49227",
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

Note:

1. When testing latency or validating configuration, make sure `metrics` is `null`.

2. When enabling metrics, the Xray-core instance needs to be run in a **child process**.

### validation

Verify the Xray configuration.

### xray

Start and stop Xray instances.

## nodep_wrapper

export nodep.

### xray_wrapper

export xray.

# Credits

[Project X](https://github.com/XTLS/Xray-core)

[VMessPing](https://github.com/v2fly/vmessping)

[FreePort](https://github.com/phayes/freeport)

# License

This repository is based on the MIT License.
