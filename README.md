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

By default, the build script does not clone [Xray-core](https://github.com/XTLS/Xray-core). It uses Go modules and pins Xray-core to release tag `v26.7.28` through its pseudo-version.
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

> [!WARNING]
> **Use only one Go runtime per process.** Go does not support loading multiple
> independently built Go runtimes into one process. Both the cgo and gomobile
> Apple artifacts embed a Go runtime. Do not link `LibXray.xcframework`
> together with another independently built Go or gomobile framework into the
> same app or extension executable. Doing so can fail at link time or crash
> during runtime initialization, before application code or
> `NEPacketTunnelProvider` runs.
> If one process needs Go packages from several frameworks, include those
> packages in the same Go build or `gomobile bind` invocation and produce one
> framework so they share a runtime. Merely repackaging or merging independently
> built frameworks is not sufficient. The containing app and a Network
> Extension are separate processes, so apply this rule independently to each
> target. See [Go #18976](https://github.com/golang/go/issues/18976#issuecomment-308505600),
> [x/mobile #15956](https://github.com/golang/go/issues/15956#issuecomment-373709423),
> and [libXray #116](https://github.com/XTLS/libXray/issues/116).

#### 1. use gomobile

Need "iOS Simulator Runtime".

This is the best choice for general scenarios. The single-runtime restriction
above still applies when linking other Go-based frameworks.

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

Depends on gcc and g++ in `PATH`.

Native amd64 and arm64 builds are supported. The release workflow builds each
architecture on its matching GitHub-hosted Windows runner.

## API

libXray exposes a single structured entrypoint:

```go
func Invoke(requestJSON string) string
```

The C export is:

```c
char* CGoInvoke(char* requestJSON);
void CGoFree(char* value);
```

`CGoInvoke` allocates its response. The caller must release every non-null
response with `CGoFree`; do not use a platform allocator directly.

The request is a JSON object:

```json
{
  "apiVersion": 2,
  "method": "runXray",
  "payload": {
    "xrayJson": "{\"outbounds\":[...]}"
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

1. Invoke currently accepts only `apiVersion: 2`. Xray configurations are
   passed as UTF-8 JSON text in `xrayJson`; libXray does not read configuration
   file paths.
2. A top-level `env` field is ignored and has no effect. Xray-core runtime
   environment options belong in the root `env` object of the Xray config.
3. `SetTunFd` has been removed. When the fd is only known at runtime, write
   `xray.tun.fd` into the Xray config root `env` object before calling
   `runXray`.
4. `countGeoData` is not backed by an Xray config, so its `datDir` is passed in
   the method payload.
5. The complete UTF-8 encoded Invoke request and response JSON envelopes are
   limited to 16 MiB. If either limit is exceeded, Invoke returns a failure
   response with `success: false`, `data: null`, and a size-limit error.
6. `convertShareLinksToXrayJson` validates each parsed outbound with the current
   Xray-core config builder. Invalid outbounds are omitted, and the method fails
   if none remain. Validation does not create or start an Xray instance.
   Its optional `age.secretKey` decrypts official age ASCII armor in memory
   before the existing parser runs. Plaintext input remains unchanged.
7. Xray-core keeps its system dialer DNS client and outbound manager in
   process-wide state. Creating another Xray instance through `pingBatch`,
   `testXray`, or the exported Go APIs while `runXray` is active may replace
   that state and affect the running
   instance. Closing the temporary instance does not restore the previous
   state. libXray does not serialize, isolate, or restore concurrent instances;
   callers that require overlapping instances must place them in separate
   processes.

Supported methods:

```text
getFreePorts
convertShareLinksToXrayJson
convertXrayJsonToShareLinks
generateAgeKeyPair
countGeoData
pingBatch
testXray
runXray
stopXray
xrayVersion
getXrayState
```

## controller

### Socket protect

Used to solve the socket protect problem on Android.

### DNS resolver

Android may expose a loopback DNS server to Go's resolver while a VPN is
active. Call `SetDNS` before `runXray` to make Go use the DNS server selected by
the VPN configuration and protect the DNS socket from the VPN tunnel. The
server must be an IP endpoint with a port, such as `8.8.8.8:53` or
`[2001:4860:4860::8888]:53`.

Call `ResetDNS` after Xray has stopped. These APIs are available only in the
Android artifact and change the process-wide Go resolver.

```java
LibXray.setDNS(controller, "8.8.8.8:53");
LibXray.invoke(runXrayRequest);

// Later, when stopping the core:
LibXray.invoke(stopXrayRequest);
LibXray.resetDNS();
```

### Process finder (per-app routing)

`ConnectivityManager.getConnectionOwnerUid()` is API 30+. On older Android
libXray falls back to parsing `/proc/net/{tcp,udp}{,6}` in pure Go.

Usage (Java/Kotlin):

```java
ProcessFinder finder = new ProcessFinder() {
    @Override
    public long findProcessByConnection(String network, String srcIP, long srcPort,
                                         String destIP, long destPort) {
        return -1; // return UID or -1
    }
};
LibXray.registerProcessFinder(finder, Build.VERSION.SDK_INT);
```

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

### age-encrypted subscriptions

`convertShareLinksToXrayJson` accepts an optional native age secret key. Only
X25519 (`AGE-SECRET-KEY-1...`) and ML-KEM-768 + X25519 hybrid
(`AGE-SECRET-KEY-PQ-1...`) identities are accepted. Recognized age armor is
decrypted in memory and limited to 16 MiB of plaintext.

```json
{
  "apiVersion": 2,
  "method": "convertShareLinksToXrayJson",
  "payload": {
    "text": "-----BEGIN AGE ENCRYPTED FILE-----\n...",
    "age": {
      "secretKey": "AGE-SECRET-KEY-1..."
    }
  }
}
```

Generate a new keypair with `keyType` set to `x25519` or `hybrid`. An omitted
`keyType` defaults to `x25519`. The `hybrid` option matches Mihomo
`age keygen-pq` and produces an `AGE-SECRET-KEY-PQ-1...` identity with an
`age1pq1...` recipient.

```json
{
  "apiVersion": 2,
  "method": "generateAgeKeyPair",
  "payload": {
    "keyType": "x25519"
  }
}
```

The response contains both `secretKey` and `publicKey`. The integrating
application must persist the pair and send only `publicKey` as
`X-Age-Public-Key`. libXray does not perform the subscription HTTP request,
persist keys, or add headers. Applications must never send the secret key over
HTTP or write decrypted subscription text to disk.

### vmess

convert VMessQRCode to Xray Json.

### xray_json

Some tools used to parse shared links.

## xray

### pingBatch

Tests multiple outbound configurations concurrently in one temporary Xray
instance. Each `xrayJson` string is parsed only for its `outbounds`; all other
root fields are ignored. The target outbound is selected by `outboundTag`, then
by the `proxy` tag, and finally by the first outbound.

```json
{
  "apiVersion": 2,
  "method": "pingBatch",
  "payload": {
    "configs": [
      {
        "xrayJson": "{\"outbounds\":[...]}"
      },
      {
        "xrayJson": "{\"outbounds\":[...]}",
        "outboundTag": "media"
      }
    ],
    "timeout": 5,
    "url": "https://cp.cloudflare.com/"
  }
}
```

Each request accepts at most five configurations and tests all accepted
configurations concurrently. Requests containing more than five configurations
fail before any configuration is tested.

The top-level response succeeds when the batch itself was accepted. Each item
has its own result; `delay` is `10000` for an error and `11000` for a timeout.
The result array has the same length and order as the input config array.
Outbound dependencies referenced by
`streamSettings.sockopt.dialerProxy` or `proxySettings.tag` are included
automatically.

### testXray

Validates an Xray configuration from the supplied JSON text without reading a
configuration file:

```json
{
  "apiVersion": 2,
  "method": "testXray",
  "payload": {
    "xrayJson": "{\"outbounds\":[...]}"
  }
}
```

### runXray

Starts the managed Xray instance from the supplied JSON text. Use `stopXray`
to stop that instance. `runXrayFromJson` is no longer a separate method.

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

[MetaCubeX age](https://github.com/MetaCubeX/age) (BSD 3-Clause)

# License

This repository is based on the MIT License.
