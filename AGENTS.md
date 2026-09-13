# libXray

Go wrapper around Xray-core for mobile and desktop clients. Keep platform-specific
App behavior out of the generic library.

## API and runtime

When changing a public API contract, read [README API](README.md#api), the
affected method section, and `invoke_model.go` / `invoke.go`. Internal-only
changes need the relevant implementation and tests, not the entire API guide.

- Keep `LibXrayAPIVersion` fixed at `3`; do not increment it within this release.
  Synchronize contract changes with typed models, downstream consumers, tests,
  and both `README.md` and `readme/README.zh_CN.md`.
- Applications use `Invoke`/`CGoInvoke` with typed requests. Config methods receive
  `xrayJson` text, not configuration file paths. Runtime `env` belongs inside
  that Xray JSON. File-oriented APIs and the desktop Core CLI retain file access.
- `TestXray` constructs an instance through `newXrayInstance` and closes it
  without calling `Start`. Callers own minimal/filtered validation configs.
  Process-level side effects are accepted; startup resources and connectivity
  remain outside this check. See [testXray](README.md#testxray) for limitations.
- Manage one running instance. Validation and temporary instances must reject
  managed-instance overlap before loading configuration and hold the lifecycle
  lock through worker completion and instance cleanup. Close temporary instances
  on every exit path; unmanaged overlaps require caller-owned process isolation.
- When changing [batch probes](README.md#pingbatch), preserve input order,
  per-item failure isolation, and raw `locationJson`; provider parsing belongs
  to the App.
- Traffic comes directly from [Xray metrics](README.md#metrics). The library
  manages the Core lifecycle only; it does not sample or persist traffic.
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

Targets and local-core options are documented in [build usage](README.md#usage).

## GitHub and reviews

- Use explicit `--repo XTLS/libXray` or repository API endpoints; the Git remote
  uses an SSH alias. Write issue/PR titles, descriptions and comments in English.
  Keep PR content self-contained without references to other repos' PRs.
- Review the PR's actual remote base/head, not unpushed local changes; record
  the commit IDs without switching the checkout. Report Standards and Spec
  separately, with severity, location, concrete impact and evidence.
- A review does not authorize edits, comments, label changes, closure or pushes.
  Check actual labels when an authorized action needs them; no triage setup is required.

## Verification

Run `git diff --check` for all changes. Match further verification to the change;
expand or repeat checks only for new changes, failures, or unresolved concerns.

- Go changes: `gofmt` changed files and run affected tests. Use
  `go test ./... -count=1` for shared lifecycle, API, or dependency changes,
  or when the impact cannot be contained to specific packages.
- Invoke changes: cover dispatch/models, response shapes, and removed methods
  where relevant; verify downstream request models against the same contract.
- Bridge/build changes: build the affected artifact where supported. Report
  unsupported targets or unbuilt artifacts explicitly.
- Documentation-only changes: check referenced paths/anchors; no Go tests or
  native builds are needed.
