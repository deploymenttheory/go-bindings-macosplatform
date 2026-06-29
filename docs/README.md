# Documentation

Guides and reference material for working on and with this macOS bindings code
generator. Start with the top-level [`README.md`](../README.md) for the project
overview and [`CLAUDE.md`](../CLAUDE.md) for the architecture and the generator
commands; the documents here go deeper on specific areas.

| Document | What it covers |
|---|---|
| [`developer_guide.md`](developer_guide.md) | A practical walkthrough for **building native macOS apps** with the bindings (app lifecycle, windows, menus, VM management, blocks). |
| [`extraction_workflow.md`](extraction_workflow.md) | How SDK headers become Go packages: the Clang AST → `.gometa.json` → Go-source three-phase pipeline. |
| [`naming.md`](naming.md) | The naming standard (the contract for generator code and generated identifiers). |
| [`metadata_overrides.md`](metadata_overrides.md) | Declarative per-framework corrections (`overrides.json`) applied at load time. |
| [`handling_variadics.md`](handling_variadics.md) | How ObjC variadic methods are bridged (and why most can't be). |
| [`opinionated_library.md`](opinionated_library.md) | The `opinionated/library/` ergonomic helpers: why they exist, what they contain, and raw-vs-opinionated comparisons. |
| [`appledocs.md`](appledocs.md) | Harvesting Apple's developer documentation into `appledocs.json` sidecars and merging it into the generated docs. |
| [`markdown-syntax-guide.md`](markdown-syntax-guide.md) | Markdown reference. |

See also the runnable [`examples/`](../examples) and their
[adoption guide](../examples/README.md) for how to use the bindings in your own
project.

## License

Distributed under the MIT License. See [`LICENSE`](../LICENSE).
