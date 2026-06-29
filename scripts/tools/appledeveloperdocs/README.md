# appledeveloperdocs

Harvests Apple's developer documentation from the **DocC render API** and writes
it into per-framework sidecar files (`appledocs.json`) next to the committed
`.gometa.json` metadata. The codegen loaders merge these into the `Doc` fields at
load time (Apple-preferred, header fallback), so the generated raw bindings
(`bindings/frameworks/`, `bindings/libraries/`) and the idiomatic layer
(`opinionated/idiomatic/`) carry Apple's prose alongside the code.

This is a Go mirror of [`csandrew-dev/AppleDocParser`](https://github.com/csandrew-dev/AppleDocParser).
Where that tool drove a Selenium browser and scraped rendered HTML for Apple's
REST-API schema pages, this fetches Apple's **structured DocC JSON** directly (no
browser, stdlib only) and targets framework symbols — classes, methods,
properties, enums, structs.

## Usage

```sh
# One framework
go run ./scripts/tools/appledeveloperdocs fetch --framework Foundation

# Several, with full Discussion text (not just the abstract)
go run ./scripts/tools/appledeveloperdocs fetch --framework Foundation,AppKit --deep

# Everything in the committed metadata tree
go run ./scripts/tools/appledeveloperdocs fetch --framework all
```

Flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `--framework` | (required) | comma-separated framework names, or `all` |
| `--metadata` | `./metadata` | metadata directory holding `.gometa.json` files |
| `--cache` | `<metadata>/.appledocs-httpcache` | HTTP response cache dir (gitignored) |
| `--deep` | `false` | also harvest each symbol's Discussion, not just the abstract |
| `--delay` | `50ms` | minimum delay between live HTTP requests |
| `--concurrency` | `5` | number of parallel fetch workers, each rate-limited by `--delay` |

## How it works

1. **Metadata-driven**: reads the committed `.gometa.json` to learn exactly which
   symbols we bind, then fetches only those pages — no blind crawling.
2. **DocC JSON**: `https://developer.apple.com/tutorials/data/documentation/<fw>/<symbol>.json`.
   A class page's `references` map embeds the abstract of every child member, so a
   single class fetch documents the whole class.
3. **Objective-C projection**: the base page is Swift-named. The tool applies the
   `occ` entry of the page's `variantOverrides` (an RFC-6902 JSON-Patch) to project
   it into the Objective-C view, where `references[].title` is the exact ObjC
   selector and `fragments[0]` is `- ` (instance) or `+ ` (class). See
   [`docc/jsonpatch.go`](docc/jsonpatch.go) and [`docc/render.go`](docc/render.go).
4. **Sidecar**: matched docs are keyed by metadata identity (class/enum names, the
   `±selector` method key, property/member names) and written to `appledocs.json`.

The HTTP cache makes re-runs (and tests) offline-friendly; negative (404)
responses are cached too.

## Scope

Harvests classes (+ methods, properties), enums (+ members), structs, and
protocols. C functions/externs are not fetched (their DocC slugs are not derivable
from the symbol name); their header doc comments are emitted as before. Rich
markup (lists, code blocks, links) is flattened to plain text for `//` comments.
