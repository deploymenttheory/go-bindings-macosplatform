# Metadata overrides

Scanned metadata is never perfect: a header misdeclares a type, Clang drops an
attribute, an API is unusable through the bridge. The override layer fixes
exactly one declaration as reviewable data — without touching scanner or
typemap code, and without editing the committed `.gometa.json` (which stays
pure observed Clang output).

## Where overrides live

One optional JSON file per framework, next to its metadata:

```
metadata/frameworks/<name>/overrides.json
metadata/libraries/<name>/overrides.json
```

Both pipeline loaders apply the file immediately after reading the
`.gometa.json`. A re-scan never deletes or modifies override files. Entries
that no longer match anything (stale after an SDK bump) print warnings during
every `generate bindings` run — prune them when you see them.

## Format

All sections are optional. JSON has no comments, so use the `comment` field
to record **why** each override exists (link the issue or the header line).

```json
{
  "comment": "FooRef is misdeclared in Foo.h line 120 (rdar://12345)",

  "exclude_classes": ["FooUnscannable"],

  "exclude_methods": [
    {"class": "FooThing", "selector": "brokenMethod:withArg:"},
    {"class": "FooThing", "selector": "factory", "class_method": true}
  ],

  "exclude_functions": ["FooDoomedFunc"],

  "remap_types": [
    {"class": "FooThing", "selector": "doIt:", "param": "value", "objc_type": "FooOptions"},
    {"class": "FooThing", "selector": "copyState", "param": "return", "objc_type": "FooState *"},
    {"function": "FooCreate", "param": "flags", "objc_type": "FooOptions"}
  ],

  "force_bitmask_enums": ["FooOptions"],

  "availability_fixes": [
    {"class": "FooThing", "macos_introduced": "11.0", "macos_deprecated": "15.0"},
    {"enum": "FooLegacyMode", "unavailable": true},
    {"function": "FooCreate", "macos_introduced": "12.0"}
  ],

  "link_lib": "foo"
}
```

### Operations

| Operation | Effect |
|---|---|
| `exclude_classes` | Drop the class entirely (no Go type, no bridge). References elsewhere degrade per the normal unresolved-type rules. |
| `exclude_methods` | Drop one selector from one class. `class_method` defaults to `false` (instance method). |
| `exclude_functions` | Drop a C function (all overloads of that name). |
| `remap_types` | Replace the ObjC type of one parameter (`"param": "<name>"`) or the return value (`"param": "return"`). The new type string goes through the normal type mapper. |
| `force_bitmask_enums` | Set `IsBitmask` so the enum emits flag-style helpers. |
| `availability_fixes` | Overwrite introduced/deprecated versions or the unavailable flag on a class, enum, or function. Only the fields you specify are changed. |
| `link_lib` | Override the `-l<lib>` linker flag for C libraries. |

## Workflow

1. Edit or create `metadata/frameworks/<name>/overrides.json`.
2. Regenerate: `go run ./cmd/generate/ bindings`.
3. Confirm the effect in `bindings/` and that no stale-entry warnings appear.
4. Commit the override file together with the regenerated bindings.

After an SDK bump, watch the `… — stale entry?` warnings: Apple may have
fixed the header, making the override unnecessary.

## Implementation

- Schema and applier for the CGo pipeline: `internal/overrides/`
- Mirrored applier for the purego pipeline (its own meta model): `internal/purecg/overrides/`
- Load points: pass 1 of both `pipeline.LoadAll` implementations.
