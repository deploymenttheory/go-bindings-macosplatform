// Package pipeline orchestrates the load and generate phases of the
// code-generation pipeline.
//
// # Loading
//
// [LoadAll] reads one or more .gometa.json files produced by the scanner and
// builds a [Registry]. The registry resolves cross-framework concerns that
// cannot be determined per-framework in isolation:
//
//   - Class ownership: the framework with the fewest non-zero methods for a
//     class name wins ("fewest methods wins" heuristic), preventing re-exported
//     declarations in umbrella headers from stealing ownership.
//   - Protocol and enum ownership: scored by member count; ties broken by
//     name-prefix match against the framework name.
//   - Superclass cycle detection: a DFS validates that no class is its own
//     ancestor before generation begins.
//
// # Generation
//
// [Generate] consumes a [Registry] and a [Config] and writes the frameworks/
// tree. Before writing it:
//
//  1. Topologically sorts frameworks by superclass dependency (Kahn's
//     algorithm) so a class's embedding type is always emitted before the
//     class that embeds it.
//  2. Detects import cycles between frameworks and breaks them by substituting
//     unsafe.Pointer for the minimum-weight set of cross-framework type
//     references ([resolveBlockedImports]).
//  3. Removes stale .go and bridge files from the output directory before
//     writing new ones.
//
// The [Config] struct controls output directory, module prefix, verbosity, and
// which frameworks to include or skip.
package pipeline
