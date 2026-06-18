# LuLu (Go port)

A port of [Objective-See's **LuLu**](https://github.com/objective-see/LuLu) macOS
firewall to Go, built on this SDK. It demonstrates the three capabilities added
for this example:

- **NEFilterDataProvider subclassing in Go** (`bindings/runtime/purego` `NewDelegate`/`RegisterClass`),
- **NSXPCConnection from Go** (`bindings/runtime/purego` `XPCProtocol`/`XPCConn`/`XPCListener`),
- **idiomatic CGo libraries** (`opinionated/idiomatic/libraries/libproc` for process attribution).

It mirrors LuLu's target layout:

| LuLu (ObjC)            | This port (Go)                                  |
|------------------------|-------------------------------------------------|
| `Shared/` (Rule, protos, consts) | `shared/`                              |
| `Extension/Rules`      | `rules/`                                        |
| `Extension/FilterDataProvider` | `extension/provider.go`                 |
| `Extension/XPCListener`/`XPCDaemon` | `extension/daemon.go`              |
| `Extension/Process`/`Binary` | `extension/process.go` (via libproc)      |
| `Extension/main.m`     | `extension/run.go` + `cmd/luludaemon`           |
| `App/`                 | `app/` + `cmd/lulu`                             |

## Data flow

```
new outbound flow
  → extension: handleNewFlow:(NEFilterFlow*)         (provider.go)
      → attribute to process via libproc proc_pidpath (process.go)
      → rules.Engine.Find(processKey, remoteAddr, port)
          allow → NEFilterNewFlowVerdict allowVerdict
          block → NEFilterNewFlowVerdict dropVerdict
          none  → default verdict (LuLu prompts the user over XPC)
app ──XPC──► daemon (XPCDaemonProtocol): get/add/delete/toggle rules
daemon ──XPC──► app (XPCUserProtocol): rulesChanged / showAlert
```

## Build

```sh
go build ./examples/lulu/...
```

Two binaries:
- `cmd/lulu` — the controlling app/CLI (pure purego).
- `cmd/luludaemon` — the network extension (links CGo `libproc`).

## Run

The **CLI control surface** runs as an ordinary binary once a daemon is reachable:

```sh
lulu list                    # list rules from the daemon (XPC)
lulu allow /usr/bin/curl     # add an allow rule for a process
lulu block /usr/bin/nc 1.2.3.4
lulu delete <key> <uuid>
lulu activate                # submit the system-extension activation request
```

The **rule engine** is plain Go and unit-testable in isolation.

## Declarative configuration (JSON or YAML)

Firewall rules can be expressed as a declarative document and reconciled onto the
firewall, `kubectl apply` style. The same schema is authored in either JSON or
YAML (format chosen by file extension, or sniffed):

```yaml
version: "1"
defaultAction: block
rules:
  - name: curl to example.com
    path: /usr/bin/curl
    action: allow
    endpoints:
      - host: example.com
        port: "443"
  - name: netcat
    path: /usr/bin/nc
    action: block
```

Apply it two ways:

```sh
lulu apply config/firewall.example.yaml              # reconcile a running daemon over XPC
LULU_CONFIG=config/firewall.example.yaml luludaemon  # daemon reconciles at startup
lulu export current.yaml                             # dump the live rules to a document
```

Reconciliation (`config` package) is:

- **Idempotent** — rule identity is content-addressed (a hash of path + action +
  host + port), so re-applying an unchanged document is a no-op; editing an
  endpoint cleanly replaces the old rule.
- **Safe** — only rules created by `apply` are marked *managed* and eligible for
  pruning. Rules added by hand (`lulu allow`/`block`) are never removed by an apply.

The engine-local path (`config.Apply` over a `rules.Engine`) is plain Go and is
covered by `config/config_test.go`; the XPC path drives the same reconciler
against the live daemon. Sample documents: `config/firewall.example.{yaml,json}`.

YAML support uses `gopkg.in/yaml.v3` — the one place this example departs from the
SDK's otherwise dependency-free posture (JSON alone is stdlib).

## What requires Apple provisioning (not doable with `go build` alone)

macOS only loads `cmd/luludaemon` as a *system extension* when it is packaged and
signed correctly — this is an OS constraint, not an SDK one:

1. The extension binary must live in a host app bundle at
   `LuLu.app/Contents/Library/SystemExtensions/com.example.lulu.extension.systemextension/`,
   with the `Info.plist` here naming `LuLuFilterDataProvider` as its
   `NSExtensionPrincipalClass` (the Go daemon registers that exact ObjC class at
   startup, subclassing `NEFilterDataProvider`).
2. Both the app and the extension must be **code-signed** with the entitlements
   in `LuLu.entitlements` / `extension/LuLuExtension.entitlements`. The
   `com.apple.developer.networking.networkextension` entitlement is **restricted**
   and must be granted to your App ID through an Apple Developer provisioning
   profile.
3. The mach service name (`shared.DaemonMachServiceName`) must match an
   `NEMachServiceName` declared in the extension's entitlements/Info.plist and be
   team-prefixed.

Without a Developer account + signing, the activation request is rejected by the
OS and the filter never loads — the firewall logic, rule engine, XPC wiring, and
process attribution are all exercised by the code here, but a live install is
gated on provisioning you control.

> Note: running a NetworkExtension principal class implemented as a purego-
> registered ObjC class is at the edge of what macOS supports; treat the live
> system-extension load as experimental. The CLI, rules engine, and XPC layer are
> the parts meant to run today.
