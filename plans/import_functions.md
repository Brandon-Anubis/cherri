# Problem statement
Cherri can already model Apple Shortcuts actions via built-in definitions and `#define action`, but there is no first-class, low-friction mechanism to install, discover, import, and maintain third-party command packs (Toolbox Pro, Nautomate, GizmoPack, etc.) across compile, decompile, search, and docs workflows.
## Current state (relevant architecture)
Action metadata is represented by `actionDefinition` and supports declarative params, enums, static extra params, and custom check/make/decomp hooks (`action.go:74`).
Built-in actions are hardcoded in `actions_std.go` (`actions_std.go:48`) and additional declarative actions are loaded from `#define action` parsing (`action.go:700`, `action.go:736`).
Standard action categories are statically listed (`actions_std.go:2006`) and loaded through include injection (`actions_std.go:2035`, `actions_std.go:2075`).
Include resolution currently supports embedded `actions/*`, `stdlib`, or local files (`includes.go:85`).
Compile-time action lookup fails on unknown symbols and only suggests standard category includes (`parser.go:1489`, `actions_std.go:2087`).
Decompiler identifier matching depends on what is loaded in `actions` and similarly only auto-inserts standard includes (`decompile.go:259`, `decompile.go:1060`, `decompile.go:1255`).
Action search/docs are oriented around built-in categories (`search.go:15`, `docs.go:32`).
## Goals
Enable reusable third-party action packs with typed function calls (not raw actions), easy project-level import, deterministic resolution, and decompile round-tripping that can recover pack imports.
Keep backwards compatibility with existing `#include` and `#define action` behavior.
Support both curated official packs and user-local/private packs.
## Non-goals (v1)
No dynamic runtime introspection of installed iOS/macOS apps.
No remote code execution or Go-plugin loading from third parties.
No requirement to encode advanced `make/check/decomp` Go hooks in external packs for initial release.
## Proposed mechanism: Function Packs
### 1) Introduce a pack manifest + declarative catalog format
Add a pack descriptor format (JSON) with strict schema versioning:
- Pack identity: `id`, `displayName`, `version`, `author`, `homepage`.
- Compatibility: minimum Cherri version, optional iOS range.
- Source files: one or more Cherri definition files containing `enum` and `#define action` blocks.
- Namespacing metadata: recommended function prefix (e.g., `tbp_`, `na_`, `gz_`) and collision policy.
- Identifier index hints: optional mapping from full Shortcut identifier to function name for faster decompile matching.
Keep action signatures in existing Cherri syntax to reuse current parser and avoid duplicating type systems.
### 2) Add a pack registry/discovery subsystem
Create a new loader layer (e.g., `function_packs.go`) that discovers manifests from:
- Repo-local path (e.g., `./function-packs/`),
- User path (e.g., `~/.config/cherri/function-packs/`),
- Optional explicit CLI path (`--pack-dir`).
For each pack: validate schema, resolve referenced files, and build in-memory indexes:
- `packByID`,
- `actionToPack` (function identifier -> pack),
- `shortcutIdentifierToPackAction` (full workflow identifier -> pack/function).
Registry load must be deterministic (stable sort by path + pack id).
Invalid discovered packs should be marked unusable and surfaced via diagnostics, but only packs explicitly imported by the current source file (or explicitly requested via CLI) should hard-fail the command.
### 3) Add explicit source-level imports for packs
Introduce a new directive, `#import 'pack-id'` (or equivalent), parsed during pre-processing before action calls are validated.
Implementation touchpoints:
- Add token for import directive (`token.go`).
- Extend pre-parse flow to process imports alongside includes (`parser.go:108`, `includes.go:28`).
- Resolve imported pack files into `lines` similarly to include expansion so `handleActionDefinitions()` can keep working unchanged (`action.go:700`).
This keeps user ergonomics simple and avoids requiring full file paths in every project.
### 4) Generalize "missing include" into "missing provider"
Refactor `checkMissingStandardInclude` (`actions_std.go:2087`) into a provider-aware resolver:
- First check local declared/imported actions.
- If unresolved, check installed packs index.
- If exactly one pack matches, emit targeted error: `Action 'x()' requires #import 'toolboxpro'`.
- If multiple packs match, emit ambiguity error with candidate pack ids and require explicit import.
Preserve current behavior for built-in `actions/*` categories.
### 5) Define precedence and collision policy
Adopt explicit resolution order:
1. Local source `#define action` declarations.
2. Imported function packs.
3. Built-in standard actions.
On name collisions between packs or with built-ins, require one of:
- Pack-specific alias import (`#import 'toolboxpro' as 'tbp'`) plus generated prefixed names, or
- Manifest-enforced prefixing.
Do not silently override existing behavior.
### 6) Decompile integration for third-party actions
Enhance decompile matching so third-party actions round-trip:
- Use pack identifier index while matching unknown `WFWorkflowActionIdentifier` (`decompile.go:1060`).
- When matched to a pack action, emit `#import 'pack-id'` at file top (similar to `popLine` include insertion path in `decompile.go:1255`).
- Retain `rawAction(...)` fallback when no pack/builtin mapping exists.
This allows imported Shortcuts containing Toolbox Pro/Nautomate/GizmoPack actions to become maintainable Cherri source instead of opaque raw actions.
### 7) CLI support for discoverability and maintenance
Extend CLI arguments (`args.go:9`) with pack lifecycle commands:
- `--list-packs` (show installed/discovered packs),
- `--pack-info=<id>`,
- `--pack-dir=<path>` (additional search root),
- Optional future: `--install-pack=<url|path>` and `--update-packs`.
Also extend `--action` search to show pack origin and availability (`search.go:15`).
### 8) Docs generation integration
Extend docs generation pipeline (`docs.go:32`) to optionally include packs:
- `--docs=<pack-id>` or `--docs-pack=<id>` to render pack-specific action docs,
- include manifest metadata (pack version/source) in header.
This makes third-party catalogs self-documenting and easy to audit.
### 9) Seed packs for popular ecosystems
Create first-party maintained starter packs for:
- Toolbox Pro,
- Nautomate,
- GizmoPack.
Each pack should ship with:
- manifest,
- declarative action file(s),
- enum definitions,
- compatibility notes,
- fixture Shortcuts or Cherri snippets for regression tests.
### 10) Testing strategy
Add tests covering:
- Manifest parsing/validation failures and conflict detection.
- Import directive parse + registry resolution.
- Compile success for third-party action calls with typed args.
- Undefined action diagnostics suggesting correct `#import`.
- Decompile recovery: unknown identifier -> pack match -> top-level import insertion.
- Backwards compatibility: existing standard action includes and tests remain green (`cherri_test.go:17`).
## File-level implementation roadmap
`token.go`: add import token.
`parser.go`: parse/import directive handling in pre-parse pipeline; keep compile-time action collection semantics intact (`parser.go:108`, `parser.go:1489`).
`includes.go`: unify include/import expansion pipeline, with separate resolvers for file includes vs pack imports (`includes.go:28`, `includes.go:85`).
`action.go`: reuse existing action definition parsing; add optional source metadata for actions (origin pack id) when loaded (`action.go:74`, `action.go:736`).
`actions_std.go`: refactor static include suggestion flow into pluggable provider resolver (`actions_std.go:2006`, `actions_std.go:2087`).
`decompile.go`: integrate pack-aware identifier mapping and import insertion (`decompile.go:259`, `decompile.go:1060`, `decompile.go:1255`).
`args.go`: register pack-related flags (`args.go:9`).
`search.go`, `docs.go`: include pack-origin-aware discovery and docs output (`search.go:15`, `docs.go:32`).
New files: pack manifest structs, schema validation, registry/index builder, and tests.
## Rollout plan
Phase 1: Internal pack registry + local filesystem packs + `#import` directive.
Phase 2: Provider-aware missing-action diagnostics + decompile import recovery.
Phase 3: CLI pack management and docs/search integration.
Phase 4: Curated official pack set (Toolbox Pro, Nautomate, GizmoPack) and versioned compatibility policy.
## Risks and mitigations
Identifier drift in third-party apps: store pack version metadata and add compatibility matrix tests.
Name collisions across ecosystems: enforce prefix/alias policy and deterministic resolver errors.
Security/trust of third-party catalogs: no executable plugin code in v1; validate manifest schema strictly; prefer local/explicit pack installation.
Maintenance burden: keep packs declarative and test-driven; avoid custom Go hooks unless a pack truly requires them.
## Acceptance criteria
A developer can install/import a third-party pack, call its actions with typed signatures, compile successfully, and decompile back to Cherri with pack imports preserved.
Existing projects using only built-in actions continue to compile/decompile without behavior changes.
