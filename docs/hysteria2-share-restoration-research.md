# Hysteria2 share restoration research

Research date: 2026-09-26. This note distinguishes restoring the former share
format from implementing every feature in the latest Hysteria specification.

## Implementation status — 2026-09-26

Implemented on `feat/restore-hysteria2-share`; the App changes remain on its
existing `dev-26.9.5-2` branch. The sections below retain the pre-implementation
research. The current supported fields and security boundaries are specified
in [parse_share](../README.md#parse_share), not by the historical implementation.

- Both URI aliases import; export uses `hysteria2://`. IPv6, userinfo escaping,
  Salamander, mixed/Base64/Age inputs and both local/remote hopping are covered.
  Official `pinSHA256` and verification bypass are rejected, not reinterpreted.
- `go test ./... -count=1` and the Hysteria2 race tests passed, including a real
  UDP reply after the five-second socket hop. Invoke API remains version 3.
- Standard Android and Apple Go builds succeeded and their artifacts were
  copied to the App. Module files were unchanged after the builds. No adjacent
  Xray-core, VCore, database schema or Pigeon API was changed.
- The App propagates Windows/Linux interface policy to the hopping mask's
  independent socket options. Its 48 affected Flutter tests and analyze passed.
- Android emulator validation used a separate application ID and the official
  Hysteria v2.12.3 server. Import, automatic queued probes, database preservation,
  URI export/re-import, native VPN start/switch/stop and actual HTTP downloads
  passed for plain, Salamander and hopping nodes. Six approximately 1 MiB
  transfers succeeded across 30 seconds of hopping; both destination ports and
  changed local ports were observed. The share UI displayed the generated URI.
- The controlled certificate used the explicit Xray `pcs` extension. Public-PKI
  interoperability, the reporter's provider, and physical iOS devices were not
  tested. The server deliberately blocked non-test destinations, so location
  lookup and background Geodata download failures in this isolated run were
  expected; they did not invalidate measured delay or interrupt the VPN.
- App copy and three website languages were updated. Hugo, documentation/SEO
  checks and all 13 SEO regression tests passed. This is not deployment evidence.

Local harnesses, server checksums, screenshots and the detailed results are
under workspace `references/hysteria2-share-restoration/`. Test VPN, isolated
package, ADB forwarding and local servers were stopped/removed after validation;
the developer's existing App database was not modified. No commit or push was
requested or performed during implementation.

## Recommendation and estimated cost

Restore Hysteria2 URI import and export as a focused share-conversion change,
not a protocol implementation or an App architecture rewrite. Keep Clash/Mihomo
and legacy VMessQrCode unsupported. Keep the existing core dependency and Invoke
API version/response shapes unchanged.

The ordinary valid-certificate path is low-cost. Correct URI edge cases,
security-policy boundaries, and live port-hopping checks make the complete
restoration low-to-medium cost. A planning estimate is **one to two focused
development days** for the scoped converter, regression coverage, App artifacts,
copy updates, and an available Android test endpoint. This is an engineering
estimate, not measured implementation time; it excludes a core TLS rewrite,
new protocol variants, release review, unavailable device checks, and platform
issues discovered by live testing.

| Area | Expected work | Cost / boundary |
| --- | --- | --- |
| libXray | Restore URI dispatch, protocol-specific parsing/export, and mask conversion; fix URL edge cases and security rejection behavior | Main implementation work; a few hundred lines rather than a new protocol stack |
| Regression coverage | Replace Hysteria2 rejection fixtures; cover both directions, Base64/Age, mixed input, and retained unsupported formats | Existing tests/helpers are reusable |
| App | Refresh native libraries; update five subscription-description translations and their tests; add Hysteria outbound import/share coverage | No database, Pigeon, or routing-model redesign indicated by the ordinary path |
| Public documentation | Update libXray EN/ZH support statements, App sharing contract, and website support descriptions | Small, but must reflect actual TLS/URI limitations |
| Device acceptance | Android import, queued probe, VPN traffic, export/re-import, and hopping; Apple build and eventual iOS device check | Parsing/build success alone is not acceptance |

## Issue and deletion history

[OneXray issue #201][issue-201] reports an App Store installation of OneXray
26.9.4 with Xray-core 26.9.9 on iOS, using Custom Routing. It contains no example
link, configuration, or log. The removed URI support matches the reported
regression, but the report does not establish whether the user's exact failure
is import, sharing, or runtime connectivity. Do not claim it has been reproduced
or fixed on iOS from this research.

The relevant libXray history is:

1. [`36351d7`][core-update] (2026-09-09) updated the core dependency to the exact
   revision still used today and migrated Hysteria2 port hopping to UDP masks.
2. [`613752d`][removal] (2026-09-09), `refactor(share): remove Clash and Hysteria2
   URI support`, removed `hy2`/`hysteria2` recognition, the outbound parser,
   share generation, TLS-default dispatch, and `share/hysteria_mask.go`.
3. The change entered main in [`50b9597`][main-release], carrying libXray tags
   `v26.9.9` and `v1.260909.0`. These are library tags, not an App release number.

Use the Hysteria2 portions of `613752d^` as a reference, not a wholesale revert.
The deletion also removed Clash/Mihomo, and later sharing work removed legacy
VMessQrCode and improved other protocols. Preserve those changes. The former
Hysteria parser/export/helper amount to roughly 200-250 implementation lines,
excluding tests; an updated restoration need not reproduce the old file layout.

Current `share/marshal_share.go` still projects native Hysteria outbounds,
`hysteriaSettings`, TLS fields, and FinalMask. The native JSON regression in
`share/marshal_share_test.go` still verifies Hysteria2 port-hopping JSON can
survive projection and build. Only the URI path was intentionally removed.
See the current [sharing implementation][current-share] and [API contract][current-readme].

## App integration scope

OneXray was inspected at `6d2e777f4fa8c23fe9cb04dc827251148b32bbed`, branch
`dev-26.9.5-2`, in the adjacent App repository.

- `lib/service/shared/share/xray_share_reader.dart` delegates standard parsing
  to `AppHostApi.convertShareLinksToXrayJson` without a Dart protocol whitelist.
- `lib/service/servers/outbound/map.dart` and `state_db.dart` preserve outbound
  maps and store them through the existing CoreConfig representation.
- `lib/service/connect/compiler.dart` clones selected outbound maps and applies
  managed tags/socket policy; it does not reconstruct Hysteria settings.
- `lib/pages/shared/share/controller.dart` already delegates standard link
  generation to `convertXrayJsonToShareLinks`.
- `prototypeSubscriptionDescription` in all five `lib/l10n/*.arb` languages,
  plus `test/pages/servers/subscription/form_test.dart`, currently says only
  VMessAEAD/VLESS links are supported. Correct that capability statement when
  support is delivered, without adding a new import UI.

Thus ordinary links should flow through clipboard/file/QR/subscription import,
storage, existing queued probes, selection, and sharing after the native library
is updated. This is a source-derived integration assessment, not device proof.
Port hopping additionally needs the socket-policy review described below.

## Source boundaries

- libXray inspected at `55cb29e214c5b77a1ba485da639a9e6b9b98adca`.
- The dependency in `go.mod` is
  `github.com/xtls/xray-core v1.260327.1-0.20260908222543-52a412d9e2f5`.
  Its cached module metadata resolves to
  `52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120`. Source inspection used
  `/Users/yiguo/go/pkg/mod/github.com/xtls/xray-core@v1.260327.1-0.20260908222543-52a412d9e2f5/`,
  not a sibling checkout.
- Official Hysteria `master` resolved to
  `e1366b173ccf5706e1e4630fe8aa654a4b574085` (2026-09-13). Its former
  `apernet/hysteria` URLs redirect to `HyNetworks/hysteria`.
  Online documentation was checked on the research date and may evolve.

## URI contract and core mapping

The [official URI specification][uri] defines `hysteria2` and `hy2`, optional
userinfo, default port 443, port lists/ranges, TLS parameters, and an optional
display-name fragment. Bandwidth and client listening modes are deliberately
excluded. `mport` and hopping-interval query parameters are not defined there;
supporting them would be a separately documented compatibility extension.

| Input | Mapping to the pinned core | Constraints |
| --- | --- | --- |
| `hysteria2://` / `hy2://` | Outbound `protocol: "hysteria"`; `settings.version: 2`; stream `network: "hysteria"`, `security: "tls"`, `hysteriaSettings.version: 2` | The core also accepts `method` as the stream selector and gives it precedence over `network`. Both version fields are required. [Core config][core-hysteria], [stream config][core-stream], [transport config][core-method]. |
| `auth@` / `username:password@` | `streamSettings.hysteriaSettings.auth` | Preserve decoded credentials, including a colon when password is present. Auth does not belong in outbound `settings`. [Official parser/generator][hy-client], [core transport][core-method]. |
| Host and single port | `settings.address`, `settings.port` | Apply 443 when omitted. URI IPv6 literals require brackets; the address value should contain the literal without authority brackets. [URI][uri], [RFC 3986 section 3.2.2][rfc-host]. |
| `sni` | `streamSettings.tlsSettings.serverName` | If absent, the Hysteria dialer uses the destination address through `tls.WithDestination`. [Dialer][core-dialer], [TLS runtime][core-tls]. |
| `obfs=salamander&obfs-password=...` | `streamSettings.finalmask.udp` entry with `type: "salamander"`, `settings.password` | No `packetSize` for ordinary Salamander. Password must be at least four bytes at connection wrapping time; a JSON build alone does not exercise that check. [Mask schema][core-mask], [Salamander][core-salamander]. |
| Multi-port authority, e.g. `host:443,5000-5002` | One base `settings.port` plus an outermost `udphop` UDP mask containing the entire port set | Details below. [Port hopping][hopping], [mask schema][core-mask]. |
| `#name` | Display metadata only | Decode/encode as a URI fragment, without changing connection fields. [URI][uri]. |

The official implementation obtains decoded values through `User.Username()`
and `User.Password()`. Decode userinfo exactly once: a literal `+` in userinfo
is not a space, and `%25` must not be decoded again. Generate userinfo with URI
userinfo escaping; query escaping has different `+` behavior. Query values such
as an obfuscation password containing `+` need `%2B`. [Official URL parser][hy-url].

Go 1.27.1's standard `net/url` rejects commas and ranges in the authority port.
Hysteria's URL fork explicitly admits those separators. A restoration therefore
needs a deliberate multi-port authority path while retaining normal escaping
and IPv6 handling. Validate malformed escapes, empty hosts, invalid port bounds,
reversed/empty ranges, and IPv6 with an omitted port. [Official URL fork][hy-url],
local `/opt/homebrew/Cellar/go/1.27.1/libexec/src/net/url/url.go:762`.

## TLS compatibility requires an explicit policy

Official Hysteria sets `InsecureSkipVerify` independently and compares
`pinSHA256` against the leaf certificate's DER SHA-256. Consequently, with
`insecure=0` or omission, normal trust/hostname validation runs before the pin
callback. Its parser accepts `strconv.ParseBool` spellings in addition to the
documented `1`/`0`. [Official TLS and URI code][hy-client],
[official TLS connection construction][hy-core-client].

| URI TLS state | Pinned-core behavior / restoration implication |
| --- | --- |
| No pin, `insecure` absent or `0` | Normal TLS verification; direct mapping. |
| No pin, `insecure=1` | Unsupported: `TLSConfig.Build` rejects `allowInsecure: true`. Do not silently drop the flag or claim the original connection semantics are supported. |
| Pin with `insecure=1` | `tlsSettings.pinnedPeerCertSha256` provides the intended check when the supplied fingerprint identifies the leaf; never emit `allowInsecure: true`. Xray can also accept a matching CA, which official Hysteria's leaf-only pin would reject. |
| Pin with `insecure` absent or `0` | Not equivalent: Xray's leaf pin bypasses ordinary trust/hostname validation. Adding `verifyPeerCertByName` does not restore it because a matching leaf returns success first. Reject or explicitly resolve this contract before claiming faithful import. |

The pinned-core field is a **hex string**, accepting colon-separated OpenSSL
hex and comma-separated pins, with each decoded hash exactly 32 bytes. The
official URI expresses a single certificate fingerprint. Do not base64-encode
it or substitute a digest of concatenated chain bytes. Export cannot faithfully turn arbitrary multiple pins,
CA pins, or other TLS policy into one official `pinSHA256` parameter.
[Core TLS builder, lines 361-378][core-security],
[core TLS verification, lines 284-405 and 561 onward][core-tls].

The current [Project X TLS documentation][xray-tls-doc] still describes
`allowInsecure` as deprecated; the pinned source rejects it. The dependency
source is authoritative for this restoration.

## Current UDP mask and hopping shape

For ordinary Salamander plus hopping, the source-supported shape is:

```json
{
  "finalmask": {
    "udp": [
      { "type": "salamander", "settings": { "password": "example-password" } },
      {
        "type": "udphop",
        "settings": {
          "mode": "intervalLocal,intervalRemote",
          "interval": 30,
          "remotePorts": "443,5000-5002"
        }
      }
    ]
  }
}
```

This object belongs under `streamSettings`. Omit the Salamander entry when
obfuscation is absent. A concrete base port is still required by the outbound;
using the first supplied port is a reasonable converter convention, while the
hopping mask chooses an initial remote port randomly.

`remotePorts` uses `conf.PortList`; `interval` uses `conf.Int32Range`, in seconds.
Set 30 explicitly to match Hysteria's default: Xray's runtime rejects either
interval bound below 5 and supplies no default. The mask manager reverses the
configured list before wrapping; `udphop` must be the last configured entry
because its wrapper requires level zero. These are the current `finalmask.udp`
fields, not transport-level `obfs`/`hopPorts` fields or a `udpmasks` JSON field.
[Schema][core-mask], [range types][core-common], [mask ordering][core-mask-runtime],
[hopping wrapper][core-hop-config], [hopping runtime][core-hop-runtime].

The official client changes the local UDP socket as well as the remote port.
Both interval modes above match that source behavior. In the pinned Xray code,
the receive goroutine is only started in the local-hop branch; using only
`intervalRemote` appears unable to receive replies. This is a source-derived
concern, not a reproduced connection failure. The wrapper also rejects
`FakePacketConn`, so upstream proxy chaining is not established by this mapping.
[Official hopping runtime][hy-hop], [Xray hopping runtime][core-hop-runtime],
[wrapper restriction][core-hop-config].

Local hopping opens replacement sockets with the mask's own `settings.sockopt`.
The App currently places its Windows/Linux interface selection in the parent
`streamSettings.sockopt`; the core builds the mask and parent socket settings
separately. Before claiming hopping works on every platform, check that the
managed interface/protection policy reaches replacement sockets as well.
This may require a small runtime-configuration adjustment, not a database or
native API redesign. Windows/Linux live verification remains outside the
current macOS-host validation boundary. [Stream builder][core-stream],
[hopping runtime][core-hop-runtime], App `compiler.dart:570`.

## Restoration scope and validation boundary

The latest official docs also include Gecko and separate realm schemes.
The pinned core has related mechanisms (`salamander.settings.packetSize` and a
`realm` mask), but their presence does not make them part of restoring the
former ordinary Hysteria2 links. Treat them as separate work. ECH is different:
the former and current TLS helper already maps `ech` to `echConfigList`, so
reuse and test that mapping rather than introducing a new mechanism. Unsupported
connection/security parameters must not be silently discarded.
[Latest URI specification][uri], [mask schema][core-mask],
[TLS builder][core-security], [pre-removal TLS helper][old-stream].

For export, preserve only configurations representable by the selected URI
contract; do not flatten extra UDP masks, remote-IP hopping, or distinct TLS
policies into a link that appears equivalent. Parser round trips and successful
core config construction are necessary checks but do not prove connectivity.

The former helper supported nonstandard `up`, `down`, `ports`, and
`hop-interval` query fields. Treat those as a compatibility decision, not as
official URI requirements. Recommended export follows the official URI contract
and does not emit client-specific bandwidth values. Keep complete unrepresentable
configurations in native Xray JSON instead of claiming a lossless standard link.

## Suggested implementation and acceptance order

1. Implement a protocol-local parser/exporter, reusing existing sharing helpers.
   Add both schemes to dispatch and export canonical `hysteria2://` links.
   Cover auth escaping, default 443, IPv4/IPv6, SNI, name-to-`tag`, Salamander,
   and standard multi-port authority handling. Avoid vendoring a whole URL fork
   only to parse port lists. Keep the current per-item filtering contract.
2. Resolve the TLS table above explicitly. Start with normal certificate
   verification; reject any security mode that cannot preserve the requested
   meaning. Do not silently turn insecure or pinned-only input into another
   TLS policy. The old helper itself never read `insecure` or `pinSHA256`, so
   restoring its code alone would not provide full official-format support.
3. Update conversion, generation, projection, Invoke, and Age tests. Include
   mixed subscriptions, empty/invalid parameters, export/re-import, and an
   assertion that Clash/Mihomo and VMessQrCode remain unsupported. Exercise
   the actual constructed mask with packet replies and a hop; the old tests
   covered construction/wrapping, not successful hopping traffic.
4. Build with the documented `build/main.py android` and `build/main.py apple go`
   paths, without `local` core replacement; copy verified artifacts to the App.
   Update App import/share regressions and capability copy. Preserve API version
   3 and `data.outbounds`.
5. Use an authorized known-good Hysteria2 endpoint or a controlled test server
   for Android end-to-end acceptance. Verify plain TLS, Salamander, TCP and UDP,
   and sustained traffic across at least one hop. Keep evidence under workspace
   `references/`. Android success does not close the iOS validation gap in #201;
   follow with an iOS device check. Do not start a macOS VPN or claim unperformed
   Windows/Linux validation.

The current Android emulator is available for the later implementation phase.
No implementation, dependency, branch, device, or GitHub state was changed in
this research. Only this research note was added; the proposed conversion has
not been exercised against an official Hysteria server.

## Baseline checks performed

On 2026-09-26, from the current libXray checkout:

```text
go test ./share -count=1
PASS

go test . -run '^TestInvokeConvertShareLinksRejectsRemovedFormats$' -count=1 -v
PASS (Clash, Hysteria2, Hy2)
```

These checks confirm the current sharing baseline, native Hysteria JSON build
coverage, and intentional `unsupported share format` results for Hysteria2 URI
input. They are not tests of a restored implementation or real VPN traffic.

[issue-201]: https://github.com/OneXray/OneXray/issues/201
[core-update]: https://github.com/XTLS/libXray/commit/36351d7530b2233f3c8d404e6f58cffda0cbc8e5
[removal]: https://github.com/XTLS/libXray/commit/613752d75f41629e8a43d043cf2e4c14592690fd
[main-release]: https://github.com/XTLS/libXray/commit/50b95979f5db551bd273165cf469e5daaf791341
[current-share]: https://github.com/XTLS/libXray/tree/55cb29e214c5b77a1ba485da639a9e6b9b98adca/share
[current-readme]: https://github.com/XTLS/libXray/blob/55cb29e214c5b77a1ba485da639a9e6b9b98adca/README.md#share
[old-stream]: https://github.com/XTLS/libXray/blob/36351d7530b2233f3c8d404e6f58cffda0cbc8e5/share/stream.go

[uri]: https://v2.hysteria.network/docs/developers/URI-Scheme/
[hopping]: https://v2.hysteria.network/docs/advanced/Port-Hopping/
[rfc-host]: https://www.rfc-editor.org/rfc/rfc3986.html#section-3.2.2
[xray-tls-doc]: https://xtls.github.io/en/config/transports/tls.html
[hy-client]: https://github.com/HyNetworks/hysteria/blob/e1366b173ccf5706e1e4630fe8aa654a4b574085/app/cmd/client.go
[hy-core-client]: https://github.com/HyNetworks/hysteria/blob/e1366b173ccf5706e1e4630fe8aa654a4b574085/core/client/client.go#L74
[hy-url]: https://github.com/HyNetworks/hysteria/blob/e1366b173ccf5706e1e4630fe8aa654a4b574085/app/internal/url/url.go
[hy-hop]: https://github.com/HyNetworks/hysteria/blob/e1366b173ccf5706e1e4630fe8aa654a4b574085/extras/transport/udphop/conn.go
[core-hysteria]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/infra/conf/hysteria.go#L13
[core-stream]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/infra/conf/transport_internet.go#L44
[core-method]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/infra/conf/transport_method.go#L752
[core-dialer]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/transport/internet/hysteria/dialer.go#L293
[core-security]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/infra/conf/transport_security.go#L299
[core-tls]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/transport/internet/tls/config.go
[core-mask]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/infra/conf/transport_finalmask.go
[core-common]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/infra/conf/common.go#L215
[core-salamander]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/transport/internet/finalmask/salamander/salamander.go#L30
[core-mask-runtime]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/transport/internet/finalmask/finalmask.go#L21
[core-hop-config]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/transport/internet/finalmask/udphop/config.go#L10
[core-hop-runtime]: https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/transport/internet/finalmask/udphop/conn.go
