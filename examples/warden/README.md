# Warden

A declarative macOS firewall, built on this SDK as an example. The firewall
policy is a JSON/YAML document applied authoritatively — the document is the
complete, enforced state of the firewall.

Its architecture is modeled on [Objective-See's **LuLu**](https://github.com/objective-see/LuLu)
(the NEFilterDataProvider filtering, rule engine, XPC control channel, and process
attribution map onto LuLu's targets), but Warden is configuration-driven rather
than interactive. It demonstrates three capabilities of this SDK:

- **NEFilterDataProvider subclassing in Go** (`bindings/runtime/purego` `rt.NewDelegate`),
- **NSXPCConnection from Go** (`bindings/runtime/purego` `XPCProtocol`/`XPCConn`/`XPCListener`),
- **idiomatic CGo libraries** (`opinionated/idiomatic/libraries/libproc` for process attribution).

Component layout (left: the LuLu target it's modeled on; right: this Go example):

| LuLu (ObjC original)   | Warden (this Go example)                        |
|------------------------|-------------------------------------------------|
| `Shared/` (Rule, protos, consts) | `shared/`                             |
| `Extension/Rules`      | `rules/`                                        |
| `Extension/FilterDataProvider` | `extension/provider.go`                 |
| `Extension/XPCListener`/`XPCDaemon` | `extension/daemon.go`              |
| `Extension/Process`/`Binary` | `extension/process.go` (via libproc)      |
| `Extension/main.m`     | `extension/run.go` + `cmd/wardend`              |
| `App/`                 | `app/` + `cmd/warden`                           |

## Data flow

```
new outbound flow
  → extension: handleNewFlow:(NEFilterFlow*)         (provider.go)
      → attribute to process via libproc proc_pidpath (process.go)
      → rules.Engine.Find(processKey, remoteAddr, port)
          allow → NEFilterNewFlowVerdict allowVerdict
          block → NEFilterNewFlowVerdict dropVerdict
          none  → defaultAllowPolicy verdict (Warden is policy-driven; where LuLu
                  would prompt the user, Warden applies the configured default)
app ──XPC──► daemon (XPCDaemonProtocol): get/add/delete/toggle rules
            (mutations rejected in config-managed mode)
```

## Build

```sh
go build ./examples/warden/...
```

Two binaries:
- `cmd/warden` — the controlling app/CLI (pure purego).
- `cmd/wardend` — the network extension (links CGo `libproc`).

## Run

The **CLI control surface** runs as an ordinary binary once a daemon is reachable:

```sh
warden list                    # list rules from the daemon (XPC)
warden allow /usr/bin/curl     # add an allow rule for a process
warden block /usr/bin/nc 1.2.3.4
warden delete <key> <uuid>
warden activate                # submit the system-extension activation request
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
warden apply config/firewall.example.yaml              # reconcile a running daemon over XPC
WARDEN_CONFIG=config/firewall.example.yaml wardend  # daemon reconciles at startup
warden export current.yaml                             # dump the live rules to a document
```

Reconciliation (`config` package) is **authoritative**: the document is the
*complete* firewall state.

- **Idempotent** — rule identity is content-addressed (a hash of path + action +
  host + port), so re-applying an unchanged document is a no-op; editing an
  endpoint cleanly replaces the old rule.
- **Authoritative / prune-all** — any rule *not* in the document is removed,
  regardless of who created it. A rule added out-of-band (interactively or by an
  XPC client) cannot survive an apply, so the policy can't be bypassed by adding
  local allow rules.

### Managed mode (enforcement)

A one-shot apply only enforces at apply-time. When the daemon is governed by a
config (`WARDEN_CONFIG`), it runs in **managed mode**, which closes the bypass
window:

- **Locked mutation surface** — the daemon *rejects* rule mutations over XPC
  (`addRule`/`deleteRule`/`toggle`). Policy can only change by editing the
  document. (This also disables the interactive alert→rule flow — the correct
  behaviour for an enforced policy.)
- **Continuous reconciliation** — the daemon re-applies the config on an interval
  (`reconcileInterval`, 60s), so drift from direct tampering of the persisted
  `rules.json` is reverted automatically.

**Tradeoff:** authoritative mode wipes ad-hoc/interactive rules on every apply.
That is what makes the policy enforceable, but it means you cannot mix a
declarative baseline with persistent interactive additions. Supporting both would
need a layered policy model (a locked base + a user overlay) — intentionally not
built here.

The engine-local path (`config.Apply` over a `rules.Engine`) is plain Go and is
covered by `config/config_test.go` (including a test that an out-of-band rule is
pruned by an authoritative apply); the XPC path drives the same reconciler.
Sample documents: `config/firewall.example.{yaml,json}`.

YAML support uses `gopkg.in/yaml.v3` — the one place this example departs from the
SDK's otherwise dependency-free posture (JSON alone is stdlib).

## What requires Apple provisioning (not doable with `go build` alone)

macOS only loads `cmd/wardend` as a *system extension* when it is packaged and
signed correctly — this is an OS constraint, not an SDK one:

1. The extension binary must live in a host app bundle at
   `Warden.app/Contents/Library/SystemExtensions/com.example.warden.extension.systemextension/`,
   with the `Info.plist` here naming `WardenFilterDataProvider` as its
   `NSExtensionPrincipalClass` (the Go daemon registers that exact ObjC class at
   startup, subclassing `NEFilterDataProvider`).
2. Both the app and the extension must be **code-signed** with the entitlements
   in `Warden.entitlements` / `extension/WardenExtension.entitlements`. The
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

## Adopting this

Warden is the worked example of **mixing binding layers on purpose**:

- It calls framework classes through the **idiomatic** wrappers — SystemExtensions
  (`SharedManager`, `ActivationRequestForExtensionQueue`, `SubmitRequest`),
  NetworkExtension (`NEFilterNewFlowVerdict` factories), Foundation
  (`NewNumberWithBool`), and the libproc **idiomatic library** for process paths.
- It drops to the **runtime** ([`bindings/runtime/purego`](../../bindings/runtime/purego),
  imported as `rt`) only for what has no typed wrapper: the `NEFilterDataProvider`
  **subclass** it registers (`rt.NewDelegate`), the **NSXPC** connection between the
  app and the daemon, and the **dispatch** main queue. The idiomatic layer wraps
  existing classes — it cannot define a new ObjC subclass or model XPC/dispatch, so
  those legitimately stay on the runtime.

The two layers meet through the public `obj` package: `obj.Wrap(id)` lifts a raw
queue pointer into an idiomatic argument, and `obj.ID(wrapper)` extracts the raw id
to return from an IMP or send over XPC. The `ADOPTION:` comments in
`app/activation.go`, `app/client.go`, `extension/provider.go`, and `shared/objc.go`
mark each of these decisions in the code.

For the cross-cutting rules — choosing a layer, finding the binding for any
framework, the boundary helpers, and signing/entitlement prerequisites — see the
[examples adoption guide](../README.md). The provisioning section above is this
app's instance of the "protected APIs need signing" rule.
