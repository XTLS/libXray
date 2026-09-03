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

Before restoring `go.mod` and `go.sum`, each build attempt writes the ignored
`build/build-metadata-<builder>.json`, where builder is `android`, `apple-go`,
`apple-gomobile`, `linux`, or `windows`. It records the libXray commit and tracked
dirty state (including temporary module edits), Go version, effective
`go list -mod=readonly -m all` output, and SHA-256 hashes of the effective module
files. Gomobile builds resolve `latest` by default; the record includes that resolved
version and the actual PATH binary's module version and `go version -m` output.
Set `LIBXRAY_GOMOBILE_VERSION` to a Go module version to pin the resolution; `resolvedVersion` still records the resolved value.
Non-gomobile builds record `gomobile: null`.

This is **build input evidence, not proof of a successful or matching artifact**:
failed builds also run this hook, and collection failures appear in `errors` or
as a warning without replacing the original build error. Consumers must check
the build command's success and the record's freshness; artifact verification
is separate. A missing or incomplete record must not be treated as verified input.

Linux and Windows builds also produce `bin/xray` or `bin/xray.exe`. This
session Core protects Go DNS lookups from the VPN route and accepts only:

```shell
xray run -dns <IP:port> -interface <name> -config <xray.json> [-runtime <runtime.json>]
```

All three options are required. `-dns` must be an IP endpoint, and `-config`
points directly to the Xray JSON configuration.
Optional `-runtime` reads the host metadata object described under "Managed
runtime accounting", without a wrapping `runtime` key; it does not replace `-config`.

> [!WARNING]
> **Use only one Go runtime per process.** Go does not support loading multiple
> independently built Go runtimes into one process. Every native libXray
> artifact embeds a Go runtime, whether it is produced through cgo or gomobile.
> Do not load libXray together with another independently built Go, cgo, or
> gomobile library in the same executable or process. Doing so can fail during
> build, link, or load, or crash during runtime initialization before
> application code runs.
> If one process needs Go packages from several libraries, include those
> packages in the same Go build or `gomobile bind` invocation and produce one
> native artifact so they share a runtime. Merely repackaging or merging
> independently built frameworks, archives, AARs, shared libraries, or DLLs is
> not sufficient. Separate OS processes may each load one Go runtime, so apply
> this rule independently to each process. See
> [Go #18976](https://github.com/golang/go/issues/18976#issuecomment-308505600),
> [golang/go#15956](https://github.com/golang/go/issues/15956#issuecomment-373709423),
> and [libXray #116](https://github.com/XTLS/libXray/issues/116).

### Android

use [gomobile](https://github.com/golang/mobile) .

### iOS && macOS

#### 1. use gomobile

Need "iOS Simulator Runtime".

This is the best choice for general scenarios. The cross-platform single-runtime
restriction above still applies when linking other Go-based libraries.

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
  "apiVersion": 3,
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

1. Invoke currently accepts only `apiVersion: 3`. Xray configurations are
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
   Xray JSON input is treated as a node source: only its root `outbounds` are
   retained, and all other root fields are ignored. The response contains only
   fields supported by libXray share links; unsupported and generated empty
   fields are omitted. Opaque XHTTP `extra` and FinalMask mask `settings` JSON
   remain unchanged.
   Its optional `age.secretKey` decrypts official age ASCII armor in memory
   before the existing parser runs. Plaintext input remains unchanged.
7. Xray-core keeps its system dialer DNS client and outbound manager in
   process-wide state. `pingBatch`, `testXray` (including `buildOnly`),
   `checkRoute`, and their exported Go entrypoints take the managed lifecycle
   lock and reject an active `runXray` instance before loading/building config.
   A batch holds the lock through all workers and temporary-core close. This
   also serializes these temporary operations with one another. Instances
   created outside the managed APIs are not detected or restored; callers
   requiring overlap with them must still use separate processes.

Supported methods:

```text
getFreePorts
convertShareLinksToXrayJson
convertXrayJsonToShareLinks
generateAgeKeyPair
countGeoData
pingBatch
testXray
checkRoute
runXray
stopXray
xrayVersion
getXrayState
```

### Configuration validation

`testXray` accepts `{"xrayJson":"...","buildOnly":true}` to load and build
the complete configuration without creating an Xray instance. This validates
the configuration structure, including TUN/WireGuard definitions, without
creating devices, listeners, log files, or background connections. The builder
can still read local GeoData/certificates and apply the root `env` to the current
process. Geodata asset declarations validate HTTPS URLs and existing local
files; their downloader/cron does not run during a build-only check.

`buildOnly` is optional and defaults to `false`, preserving the existing
`testXray` create-and-close behavior. That behavior does not call `Start`, but
constructors can create TUN devices, open logs, or initiate background work.
Use build-only validation for an unstarted draft; a successful build does not
prove runtime resources are available or that an instance can start. Runtime
construction/start errors remain the caller's responsibility to handle.

### Configuration URL probe

`testXray` also accepts `url`, `timeout` (1–60 seconds), and optional
`inboundTag` with `xrayJson`. It returns `data: {"delay": 12}` in integer
milliseconds. `url` and `buildOnly: true` are mutually exclusive. Omit `url`
to retain the existing validation response.

The probe sends an HTTP HEAD using the draft's DNS, routing and outbounds,
without forcing one outbound. It uses the route check's safe construction:
inbounds/log output/webhooks are disabled, WireGuard and VLESS reverse are
rejected, and the temporary instance is never started or published. It does
not test extra listeners, startup-only integrations, or every destination.
The lifecycle lock and managed-instance overlap rejection still apply.

### Draft route checking

`checkRoute` is additive to API version 3. It accepts a complete draft in
`xrayJson` and calls the pinned Xray-core Router, without starting the temporary
instance or dispatching traffic to the supplied target:

```json
{
  "apiVersion": 3,
  "method": "checkRoute",
  "payload": {
    "xrayJson": "{\"outbounds\":[{\"tag\":\"direct\",\"protocol\":\"freedom\"}]}",
    "domain": "example.com",
    "port": 443,
    "network": "tcp",
    "inboundTag": "tunIn",
    "timeout": 5000
  }
}
```

Supply exactly one of `domain` (hostname, not URL) or `ip` (IPv4/IPv6 without a
zone). `port` is 1–65535, `network` is `tcp` or `udp`, and required `timeout` is
1–60000 milliseconds. `inboundTag` is optional; omitted means an empty tag, not
an assumed VPN inbound. The existing 16 MiB envelope limit applies.

Successful `data` always includes all five fields:

```json
{"matched":false,"ruleTag":"","outboundTag":"direct","balancerTag":"","defaulted":true}
```

`matched` and `ruleTag` describe the initial Router match, preserving an empty
or duplicated original rule tag. `defaulted` means the initial Router found no
matching rule. The outbound manager then supplies the actual default outbound;
a loopback's native inbound-tag/skip-DNS transition is checked again through the
Router. `outboundTag` is the terminal selected outbound, and `balancerTag` is
the last balancer encountered, or empty when none was used. Thus a draft with a
default loopback may resolve to its configured balancer, whereas an arbitrary
Raw JSON draft is never assumed to default to `proxy`. Rules reached only after
a default loopback are not reported as initial user matches. Missing handlers, loopback cycles,
traffic-dependent loopback sniffing, and routing/selection failures return an
error instead of invented evidence. Selection uses a fresh instance, not live
balancer health/history, and does not test connectivity or the exit IP.

Only the in-memory check configuration removes inbounds, disables file/log
output, and removes rule webhooks; the caller's draft is not rewritten.
WireGuard outbounds are rejected because construction can create a TUN device
even without `Start`; VLESS reverse outbounds are rejected because construction
starts background connections. There are no inbound listeners or background probes.
DNS resolution may still send network queries through the draft configuration.
The timeout context reaches the core; a timed-out lookup never returns success.
Some core resolvers, notably `localhost`, do not honor cancellation immediately,
so this is not a strict wall-clock limit. The call waits for matching to finish
before closing the instance; it never leaves matching using an already-closed
core in the background.

`checkRoute` rejects a managed `runXray` instance in the same process and holds
the managed lifecycle lock through construction, matching, and close. The same
managed-overlap guard also applies to `testXray` (including `buildOnly`) and
`pingBatch`. It does not detect externally created unmanaged instances; callers
must still use independent execution processes when those can overlap.

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

libXray stores outbound names in `tag`. `sendThrough` keeps its native Xray
meaning as the local bind address.

### clash_meta

Parse Clash.Meta configuration.

### generate_share

convert Xray Json to VMessAEAD/VLESS sharing protocol.

### parse_share

convert VMessAEAD/VLESS sharing protocol to Xray Json.

convert VMessQRCode to Xray Json.

#### Optional parsing counts

Set `payload.includeStats: true` on `convertShareLinksToXrayJson` to return
`data: {"config":{"outbounds":[...]},"usableCount":2,"failedCount":1}`.
Omitting it (or setting it to `false`) preserves the original `data.outbounds`
response and conversion behavior.

Counts describe this input only, not added/changed nodes. Each root JSON
`outbounds` element or YAML `proxies` element is one candidate. In detected
share-link lists, each URI-like row is one candidate; blank lines, comments and
text headers are ignored. Base64 and age wrappers use the inner format's
candidates. Stats mode skips malformed individual elements without discarding
other valid elements. `usableCount` equals the final projected, buildable
outbound count; parse, build and unsupported-projection failures count toward
`failedCount`. No per-node hash comparison or deduplication is performed.

A recognized container with zero usable nodes returns `success: false` with
structured counts and `config: {"outbounds":[]}`. An unrecognized format,
malformed whole document, invalid container or decryption failure returns
`data: null`; counts are not guessed. Error text never includes rejected
candidates or decrypted subscription text. Callers must not import/replace a
subscription when no usable nodes remain.

### age-encrypted subscriptions

`convertShareLinksToXrayJson` accepts an optional native age secret key. Only
X25519 (`AGE-SECRET-KEY-1...`) and ML-KEM-768 + X25519 hybrid
(`AGE-SECRET-KEY-PQ-1...`) identities are accepted. Recognized age armor is
decrypted in memory and limited to 16 MiB of plaintext.

```json
{
  "apiVersion": 3,
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
  "apiVersion": 3,
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
  "apiVersion": 3,
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
    "url": "https://cp.cloudflare.com/",
    "locationUrl": "https://ip-check-perf.radar.cloudflare.com/"
  }
}
```

Each request accepts at most five configurations and tests all accepted
configurations concurrently. Requests containing more than five configurations
fail before any configuration is tested.

The top-level response succeeds when the batch itself was accepted. Each item
has its own result; `delay` is `10000` for an error and `11000` for a timeout.
`delay` is always present, including a successful zero-millisecond result.
The result array has the same length and order as the input config array.
Outbound dependencies referenced by
`streamSettings.sockopt.dialerProxy` or `proxySettings.tag` are included
automatically.

`locationUrl` is optional and must be an absolute HTTP(S) URL. When omitted,
no location request is made and no location fields are returned. When supplied,
each prepared item sends its latency HEAD and then its location GET using the
same client forced through that item's selected outbound and dependencies.
Each request has the configured timeout (so an item may take up to twice it).
Location time is not included in `delay`, and the two results are independent:
`success`, `delay` and `error` describe latency only; a location failure does not
invalidate a successful latency result, and GET is still attempted after a
latency failure.

A successful GET adds `location: {"ip":"203.0.113.1","countryCode":"JP"}`.
The provider must return HTTP 200 and at most 64 KiB of JSON containing
Cloudflare's `ip_address`/`country` pair or normalized `ip`/`countryCode`.
The IP must be valid and the country code must be two ASCII letters (returned
uppercase). An invalid/missing location instead adds `locationError`; errors
do not echo the URL, credentials or response body. Invalid outbound configs
retain their ordinary per-item failure and do not perform either request.

### testXray

Validates an Xray configuration from the supplied JSON text without reading a
configuration file:

```json
{
  "apiVersion": 3,
  "method": "testXray",
  "payload": {
    "xrayJson": "{\"outbounds\":[...]}"
  }
}
```

### runXray

Starts the managed Xray instance from the supplied JSON text. Use `stopXray`
to stop that instance. `runXrayFromJson` is no longer a separate method.

### Managed runtime accounting

`runXray.payload.runtime` is optional API v3 host metadata. Omitting it
preserves the original lifecycle and writes no runtime snapshots. Hosts opt in
with this object (also the complete content of the desktop `-runtime` file):

```json
{
  "statePath": "/private/app/run/runtime.json",
  "planId": "opaque-plan-id",
  "inboundTag": "tunIn",
  "listen": "127.0.0.1:49228",
  "token": "538fc3253a3e433491bc2d653fc74214"
}
```

The host supplies an existing private directory and an absolute `statePath`.
`planId` and `inboundTag` must be nonempty and at most 256 bytes. `planId` is
opaque and must not contain credentials. Metadata stays separate from Xray
JSON, so user configuration cannot override it. The named inbound must exist,
with uplink/downlink system statistics and a statistics manager enabled.
`listen` and `token` may both be omitted to save snapshots without HTTP. When
enabled, `listen` must be `127.0.0.1:<port>` with port 1–65535, and the host must
generate a fresh random 32-character lowercase hex `token`. Keep it private;
do not reuse the example token. Invalid metadata, an occupied HTTP port, corrupt
saved state, an archive failure, or an initial save failure rejects startup;
any constructed core and statistics listener are closed.

The saved file contains only the current session's raw inbound counter values:

```json
{
  "version": 1,
  "session": {
    "id": "2a7e2e49b947a802d8b39af4fbc48f52",
    "planId": "opaque-plan-id",
    "startedAtMs": 1788300000000,
    "endedAtMs": 0,
    "uplink": 120,
    "downlink": 800
  },
  "available": true,
  "sampledAtMs": 1788300030000,
  "savedAtMs": 1788300030000,
  "error": ""
}
```

Timestamps are Unix milliseconds. Each new start generates a random
32-character lowercase hex session ID, even when replaying identical metadata.
`endedAtMs: 0` means no final stop was saved; it is not proof that the VPN is
running. The host saves an initial snapshot, samples/saves every 30 seconds,
and attempts a final sample/save before closing the core on `stopXray`.

Sampling reads the named inbound's `Value()`, never resets it and never adds
outbound/node counters. Repeated samples do not accumulate bytes. A nonnegative
counter rollback is recorded as the smaller raw value, not a synthetic delta.
Missing or negative counters set `available: false` and
`error: "counters_unavailable"`, retaining the last valid nonnegative values.
Idle valid counters report available zero. There are no application-wide totals,
reset generations, or VPN control HTTP methods.
`resetRuntime` is not an Invoke method. Applications may read existing Xray
metrics for live rates; their own totals/reset policy stays outside libXray.

Before a new session can replace `runtime.json`, the previous valid saved
snapshot is atomically archived beside it as
`runtime-sessions/<session-id>.json`. The archive preserves its raw counters,
timestamps, and any unset ending; it does not infer missing traffic or a crash
time. Repeated unsuccessful starts reuse the same archive filename. A failed
preparation may therefore leave the same session in both current and archive;
consumers must identify sessions by ID, not count files. Archives are retained
until the App explicitly acknowledges them; their counters are never carried
into the new session.
The archive directory rejects symlinks/non-directories and is created mode 0700.

Snapshot files use a mode-0600 same-directory temporary file, sync, and atomic
replacement (Windows uses `MoveFileEx` with replace-existing and write-through).
The private parent directory/Windows ACL remains the host's responsibility.
Failed saves leave the previous complete disk snapshot for later retry; a final
save error is returned but never prevents core shutdown. An error after rename
can have an uncertain persistence outcome, so consumers must re-read saved
snapshots through HTTP when available.
This is reference data, not billing: crashes/forced termination can lose the
tail after the last successful save, with no strict 30-second loss bound. A
restart archives only that last saved tail and does not fabricate final values.

A nonblocking OS lock on `statePath + ".lock"` is held until core close,
preventing another process from writing the same current/archive sequence.
Hosts must use one consistent canonical path and leave the lock file in place.
App code reads snapshots through HTTP instead of opening the host's files, so
macOS System Extension files can remain root-owned. This does not provide
graceful final settlement when Windows forcibly terminates a job.

#### Snapshot HTTP

The optional statistics listener starts with the managed session and closes on
stop, including when the final save fails. It uses a separate loopback port
from Xray's native metrics; it provides no VPN start/stop/configuration methods.
Every request requires `Authorization: Bearer <token>`. Responses use
`Cache-Control: no-store`; CORS is not enabled.

- `GET /runtime` returns `{"current": <snapshot-or-null>, "archived": [<snapshot>, ...]}`.
- `POST /runtime/ack` accepts `{"removeSessionIds": ["<session-id>", ...]}` and
  returns the same shape with the remaining archives. IDs must be 32-character
  lowercase hex. Unknown IDs are harmless, repeated acknowledgment is safe,
  and the current session is never deleted, including its duplicate archive.

Requests read the host's saved atomic snapshots, without sampling, resetting
counters, or updating the save time. Use native metrics for live rates. The App
must durably save its totals/session watermarks **before** acknowledging archives;
if the HTTP request fails, retain the watermarks and retry. Failed deletions
remain in the returned archive list. Only valid snapshot files immediately in
the host's own `runtime-sessions` directory are eligible; request paths and
symlinks are rejected. A corrupt snapshot makes the request fail rather than
silently lose accounting data.

Acknowledgment bodies are limited to 64 KiB and responses to 16 MiB; requests
have bounded read/write timeouts. There is no pagination. If unacknowledged
archives exceed the response limit, the request fails and the App retains its
last known data. While stopped, HTTP is unavailable: the App can show its own
persisted last-known snapshot/totals, retain reset watermarks, and reconcile
saved tails and archives on the next connection. libXray never owns App totals
or clear/reset policy.

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
