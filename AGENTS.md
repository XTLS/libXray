# Project Overview

libXray is a Go wrapper around Xray-core for mobile and desktop applications.
It exposes one structured JSON entrypoint, provides share-link and GeoData
utilities, and builds native artifacts for Android, Apple platforms, Linux, and
Windows.

The public Invoke contract is intentionally small. Platform applications should
construct typed request models, serialize them at the native boundary, and call
`Invoke` or `CGoInvoke`. Do not add platform-specific application behavior to
the generic API.

# Repository Layout

| Path | Purpose |
| --- | --- |
| `invoke.go` | Invoke request validation, method dispatch, and response encoding. |
| `invoke_model.go` | Public method enum and typed request/response models. |
| `xray/` | Xray instance lifecycle, configuration validation, and batch latency testing. |
| `share/` | Share-link parsing, validation, and generation. |
| `geo/` | GeoData inspection helpers. |
| `controller/` | Android socket protection and process lookup integration. |
| `dns/` | Android VPN-aware process DNS resolver. |
| `memory/` | Platform-specific memory-pressure handling. |
| `nodep/` | Small utilities that do not depend on the managed Xray instance. |
| `cgo_bridge/` | C ABI exports used by Apple, Linux, Windows, and Dart FFI. |
| `android_wrapper.go` | Android-only gomobile interfaces and controller registration. |
| `build/` | Cross-platform build scripts and artifact assembly. |
| `.github/workflows/` | CI builds and release artifact publication. |
| `README.md` | English integration documentation. |
| `readme/README.zh_CN.md` | Chinese integration documentation. |

# Invoke API Contract

The current API version is `2`. Requests using an omitted or different
`apiVersion` are rejected.

```json
{
  "apiVersion": 2,
  "method": "runXray",
  "payload": {
    "xrayJson": "{\"outbounds\":[...]}"
  }
}
```

Every response uses the same envelope:

```json
{
  "success": true,
  "data": {},
  "error": ""
}
```

`data` must be a JSON object for successful methods that return data, `{}` for
successful methods without data, or `null` for failures without structured
failure data. Do not return scalar values directly from `data`.

Supported methods:

- `getFreePorts`
- `convertShareLinksToXrayJson`
- `convertXrayJsonToShareLinks`
- `generateAgeKeyPair`
- `countGeoData`
- `pingBatch`
- `testXray`
- `runXray`
- `stopXray`
- `xrayVersion`
- `getXrayState`

Age-encrypted subscription support is part of the share boundary. libXray owns
native key generation, in-memory armor decryption, and parsing. Integrating
applications own HTTP headers, persistence of both generated keys, and refresh
behavior. Never log age secret keys, decrypted subscription text, or complete
Invoke requests containing those values.

`pingBatch`, `testXray`, and `runXray` receive serialized Xray configuration
text through `xrayJson`. They must not accept or read an application-provided
configuration file path. `countGeoData` is the exception because it operates on
GeoData files directly and receives `datDir` in its payload.

The complete UTF-8 Invoke request and response envelopes are limited to 16 MiB.
`pingBatch` accepts at most five configurations. It parses only `outbounds`,
ignores other root fields, and includes outbound dependencies referenced by
`streamSettings.sockopt.dialerProxy` or `proxySettings.tag`.

# Runtime Semantics

`runXray` manages one package-level Xray instance. A second `runXray` call fails
until `stopXray` closes the current instance.

`testXray` and `pingBatch` create temporary Xray instances. Xray-core contains
process-wide state, including the system dialer DNS client and outbound manager.
Running temporary instances while another Xray instance is active may replace
that state, and closing a temporary instance does not restore it. libXray does
not serialize or isolate these calls. Integrators that require independent
concurrent instances must place them in separate processes.

Xray runtime environment values belong in the root `env` object of `xrayJson`.
A top-level `env` field on the Invoke request is ignored. Missing root env fields
are governed by Xray-core behavior.

# Platform Integration

## C ABI

`cgo_bridge/main.go` exports:

```c
char* CGoInvoke(char* requestJSON);
void CGoFree(char* value);
```

`CGoInvoke` returns C-allocated memory. Every non-null response must be released
exactly once with `CGoFree`. Do not release it with a platform allocator or from
Go directly.

## Android

Android uses gomobile and produces `libXray.aar` plus
`libXray-sources.jar`. Android-only APIs include socket protection, process
lookup registration, `SetDNS`, and `ResetDNS`.

`SetDNS` changes Go's process-wide resolver and requires a protected IP endpoint
such as `8.8.8.8:53`. Call `ResetDNS` after the managed Xray instance stops.
Keep Android-only code behind the `android` build tag.

## Apple Platforms

The CGo build produces `LibXray.xcframework` for iOS, iOS Simulator, macOS,
tvOS, and tvOS Simulator. Swift callers use `CGoInvoke` and `CGoFree`; the Xray
configuration and runtime TUN fd are supplied by the application through the
typed JSON contract.

## Linux and Windows

Linux produces `linux_so/libXray.so`; Windows produces
`windows_dll/libXray.dll`. Both artifacts expose the C ABI. libXray does not
provide or manage a desktop executable wrapper.

# Building

Build scripts use the Xray-core version pinned by `go.mod` by default:

```shell
python3 build/main.py android
python3 build/main.py apple go
python3 build/main.py linux
python build/main.py windows
```

Apple also has a gomobile build path:

```shell
python3 build/main.py apple gomobile
```

To test an adjacent Xray-core checkout, place it at `../Xray-core` and append
`local`:

```shell
python3 build/main.py android local
python3 build/main.py apple go local
```

The build scripts temporarily adjust the Go module graph and restore `go.mod`
and `go.sum` when the build finishes. Generated native artifacts, downloaded
GeoData, and intermediate build directories are ignored by Git. Do not edit
generated headers, archives, frameworks, AARs, JARs, DLLs, or shared libraries
manually.

# Development Rules

1. Keep `Invoke` as the single cross-platform API entrypoint. Platform-only
   controller APIs must remain isolated by build tags.
2. Define request and response fields as typed Go models in `invoke_model.go`.
   Do not pass unstructured maps into package business logic.
3. Treat method names, JSON keys, response shapes, and `apiVersion` as a public
   wire contract. Breaking changes require an API version increment and
   synchronized integration documentation.
4. Xray configuration APIs accept `xrayJson` text, not file paths. File access
   remains limited to APIs whose purpose is operating on files.
5. Keep the English and Chinese README API sections synchronized.
6. Preserve per-item ordering in `pingBatch`; one invalid configuration should
   produce an item failure without discarding other accepted items.
7. Close every temporary Xray instance on success and error paths. Do not add
   hidden serialization or state restoration that changes existing runtime
   semantics.
8. Do not modify the adjacent Xray-core checkout as part of a libXray change
   unless the task explicitly requires it.
9. Use `gofmt` for Go source and keep changes narrowly scoped to the owning
   package.

# Validation

Run the checks appropriate to the change scope:

```shell
gofmt -w <changed-go-files>
go test ./... -count=1
git diff --check
```

Changes to build scripts or platform bridges should also build the affected
artifact. Changes to the Invoke wire contract must include dispatch/model tests,
unknown or removed method tests where relevant, response-shape tests, and
synchronized consumer model updates in downstream applications.
