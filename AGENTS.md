# libXray

Go wrapper around Xray-core for mobile and desktop clients. Keep platform-specific
App behavior out of the generic library.

## API and runtime

Before changing a method, read [README API](README.md#api), its method section,
and the models/dispatch in `invoke_model.go` and `invoke.go`.

- Keep `LibXrayAPIVersion` fixed at `3`; do not increment it within this release.
  Synchronize contract changes with typed models, downstream consumers, tests,
  and both `README.md` and `readme/README.zh_CN.md`.
- Applications use `Invoke`/`CGoInvoke` with typed requests. Config methods receive
  `xrayJson` text, not configuration file paths. Runtime `env` belongs inside
  that Xray JSON. File-oriented APIs and the desktop Core CLI retain file access.
- `TestXray` only loads/builds configuration with `core.LoadConfig`; it neither
  constructs nor starts an instance. Success does not guarantee startup or
  connectivity. Builders may still read local assets/certificates and apply `env`.
- Manage one running instance. Validation and temporary instances must reject
  managed-instance overlap before loading configuration and hold the lifecycle
  lock through worker completion and instance cleanup. Close temporary instances
  on every exit path; unmanaged overlaps require caller-owned process isolation.
- When changing [batch probes](README.md#pingbatch), preserve input order,
  per-item failure isolation, and raw `locationJson`; provider parsing belongs
  to the App.
- Before changing persistence or HTTP access, read
  [managed runtime accounting](README.md#managed-runtime-accounting).
  Save only the current session's inbound counters. Native metrics provides live
  readings; runtime HTTP provides saved snapshots. The App owns totals and reset.
- When changing [age subscriptions](README.md#age-encrypted-subscriptions),
  keep key generation/decryption in libXray and HTTP/persistence in the App.
  Never log secret keys, decrypted subscriptions, or requests containing them.

## Native integration and builds

Before changing platform bridges or build scripts, read [build](README.md#build)
and the relevant platform/controller section in README.

- Free each non-null `CGoInvoke` response exactly once with `CGoFree`.
  Load only one independently built Go runtime per process.
- Keep Android-only APIs behind the `android` build tag. When changing DNS
  integration, read [DNS resolver](README.md#dns-resolver): `SetDNS` affects the
  process resolver and `ResetDNS` follows managed-instance shutdown.
- Use `build/main.py` to generate native artifacts; do not edit generated
  headers, archives, or binaries. Verify temporary module edits are restored
  after a build and check the build command's success and resulting artifacts.
- Modify an adjacent Xray-core checkout only when explicitly requested.

Common builds:

```sh
python3 build/main.py android
python3 build/main.py apple go
```

Other targets and local-core options are documented in [build usage](README.md#usage).

## Verification

Run `git diff --check` for all changes. Match further verification to the change;
expand or repeat checks only for new changes, failures, or unresolved concerns.

- Go changes: format changed files with `gofmt`, then `go test ./... -count=1`.
- Invoke changes: cover dispatch/models, response shapes, and removed methods
  where relevant; verify downstream request models against the same contract.
- Bridge/build changes: build the affected artifact where supported. Report
  unsupported targets or unbuilt artifacts explicitly.
- Documentation-only changes: check referenced paths/anchors; no Go tests or
  native builds are needed.
