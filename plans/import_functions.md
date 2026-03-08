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
## Terminology note
The existing codebase already uses **import** in two places:
- The `--import` CLI flag: converts an iCloud-linked or file-based Shortcut to Cherri (defined in `args.go:51`).
- The `#question` directive (also referred to as "Import Questions" in some docs, a term that should be updated to "Install Questions" to reduce confusion): prompts users for values at Shortcut install time (`token.go:30`).

To avoid ambiguity with both of these, the new pack directive proposed in this plan is named **`#pack`** instead of `#import`.  All references below use `#pack 'pack-id'`.

## Goals
Enable reusable third-party action packs with typed function calls (not raw actions), easy project-level import, deterministic resolution, and decompile round-tripping that can recover pack imports.
Keep backwards compatibility with existing `#include`, `#define action`, `--import`, and `#question` behavior.
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

**Duplicate pack id handling:** Before inserting into `packByID`, check whether that id is already registered from a different root.  If a duplicate is found, do **not** silently shadow the earlier installation; instead:
- Record **all** candidate paths against the id.
- At `#pack` resolution time, hard-fail with a diagnostic listing every conflicting path and instructing the user to disambiguate via `--pack-dir` or by removing one installation.
- Only allow the same pack id from the same resolved path (content-hash equality) to succeed silently (idempotent re-install case).
### 3) Add explicit source-level imports for packs
Introduce the new directive **`#pack 'pack-id'`** (deliberately distinct from the `--import` CLI flag and `#question`), parsed during pre-processing before action calls are validated.

**Version pinning:** The directive supports an optional version constraint:
```
#pack 'toolboxpro'           // any installed version
#pack 'toolboxpro' >= '3.0'  // minimum version
#pack 'toolboxpro' == '3.1.2' // exact pin
```
An auto-generated **`cherri.lock`** file (similar to `go.sum` / `package-lock.json`) records the exact resolved version and content-hash of every `#pack` declaration.  It is created on the first successful build that references any `#pack` and should be committed to version control.  Adding a new `#pack` directive also regenerates the file (the build succeeds and updates the lock).  On subsequent builds the lockfile is checked first and the build fails if the installed pack does not match the locked version, ensuring identical behavior across machines.  `--update-packs` explicitly refreshes all locked versions to the latest installed versions.

Implementation touchpoints:
- Add token for pack directive (`token.go`).
- Extend pre-parse flow to process `#pack` alongside includes (`parser.go:108`, `includes.go:28`).
- Resolve imported pack files into `lines` similarly to include expansion so `handleActionDefinitions()` can keep working unchanged (`action.go:700`).
This keeps user ergonomics simple and avoids requiring full file paths in every project.
### 4) Generalize "missing include" into "missing provider"
Refactor `checkMissingStandardInclude` (`actions_std.go:2087`) into a provider-aware resolver:
- First check local declared/imported actions.
- If unresolved, check installed packs index.
- If exactly one pack matches, emit targeted error: `Action 'x()' requires #pack 'toolboxpro'`.
- If multiple packs match, emit ambiguity error with candidate pack ids and require explicit `#pack`.
Preserve current behavior for built-in `actions/*` categories.
### 5) Define precedence and collision policy
Adopt explicit resolution order:
1. Local source `#define action` declarations.
2. Imported function packs (in `#pack` directive order).
3. Built-in standard actions.
On name collisions between packs or with built-ins, require one of:
- Pack-specific alias import (`#pack 'toolboxpro' as 'tbp'`) plus generated prefixed names, or
- Manifest-enforced prefixing.
Do not silently override existing behavior.
### 6) Decompile integration for third-party actions
Enhance decompile matching so third-party actions round-trip:
- Use pack identifier index while matching unknown `WFWorkflowActionIdentifier` (`decompile.go:1060`).
- When matched to a pack action, emit `#pack 'pack-id'` at file top via the `popLine` helper (`decompile.go:1052`).
- Retain `rawAction(...)` fallback when no pack/builtin mapping exists.

**Colliding action names during decompile:** A bare `#pack 'pack-id'` is insufficient when two installed packs declare the same function name.  When the decompiler detects that a recovered function name exists in more than one imported pack, it must:
- Emit `#pack 'pack-id' as 'prefix'` (alias form) for all colliding packs, where `prefix` defaults to the pack id with non-alphanumeric characters removed (e.g., `toolboxpro` → `tbp`, `nautomate` → `na`).
- Rewrite every recovered call site to use the prefixed form using the same `<alias>_<functionName>` convention (e.g., `tbp_someAction(...)`); this naming convention is consistent with the manifest-enforced prefix field defined in Section 1.
- Add a diagnostic warning listing the conflict so the developer can choose a stable alias.
This allows imported Shortcuts containing Toolbox Pro/Nautomate/GizmoPack actions to become maintainable Cherri source instead of opaque raw actions.
### 7) CLI support for discoverability and maintenance
Extend CLI arguments (`args.go:9`) with pack lifecycle commands:
- `--list-packs` (show installed/discovered packs),
- `--pack-info=<id>`,
- `--pack-dir=<path>` (additional search root),
- `--update-packs` (re-resolve versions and regenerate `cherri.lock`),
- Optional future: `--install-pack=<url|path>`.
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
- `#pack` directive parse + registry resolution.
- Compile success for third-party action calls with typed args.
- Undefined action diagnostics suggesting correct `#pack`.
- Decompile recovery: unknown identifier -> pack match -> top-level `#pack` insertion.
- Decompile collision: two packs with same function name -> alias directive + prefixed call sites.
- Duplicate pack id across discovery roots -> hard-fail diagnostic.
- Version-pin enforcement via `cherri.lock` (mismatch -> fail; match -> pass).
- Backwards compatibility: existing standard action includes and tests remain green (`TestCherri()` line 20, `TestDecomp()` line 67 in `cherri_test.go`).
## File-level implementation roadmap
`token.go`: add `#pack` token.
`parser.go`: parse `#pack` directive handling in pre-parse pipeline; keep compile-time action collection semantics intact (`parser.go:108`, `parser.go:1489`).
`includes.go`: unify include/pack expansion pipeline, with separate resolvers for file includes vs pack imports (`includes.go:28`, `includes.go:85`).
`action.go`: reuse existing action definition parsing; add optional source metadata for actions (origin pack id) when loaded (`action.go:74`, `action.go:736`).
`actions_std.go`: refactor static include suggestion flow into pluggable provider resolver (`actions_std.go:2006`, `actions_std.go:2087`).
`decompile.go`: integrate pack-aware identifier mapping and `#pack` insertion via `popLine` (`decompile.go:1052`, `decompile.go:1060`); add collision-aware alias rewriting.
`args.go`: register pack-related flags (`args.go:9`).
`search.go`, `docs.go`: include pack-origin-aware discovery and docs output (`search.go:15`, `docs.go:32`).
New files: pack manifest structs, schema validation, registry/index builder, lockfile (`cherri.lock`) read/write, and tests.
## Rollout plan
Phase 1: Internal pack registry + local filesystem packs + `#pack` directive + `cherri.lock` generation.
Phase 2: Provider-aware missing-action diagnostics + decompile `#pack` recovery + collision alias rewriting.
Phase 3: CLI pack management (`--list-packs`, `--update-packs`) and docs/search integration.
Phase 4: Curated official pack set (Toolbox Pro, Nautomate, GizmoPack) and versioned compatibility policy.
## Risks and mitigations
Identifier drift in third-party apps: store pack version metadata and add compatibility matrix tests.
Name collisions across ecosystems: enforce prefix/alias policy and deterministic resolver errors.
Security/trust of third-party catalogs: no executable plugin code in v1; validate manifest schema strictly; prefer local/explicit pack installation.
Maintenance burden: keep packs declarative and test-driven; avoid custom Go hooks unless a pack truly requires them.
## Acceptance criteria
A developer can install/import a third-party pack, call its actions with typed signatures, compile successfully, and decompile back to Cherri with `#pack` directives preserved (including pinned versions from `cherri.lock`).
Existing projects using only built-in actions continue to compile/decompile without behavior changes.
