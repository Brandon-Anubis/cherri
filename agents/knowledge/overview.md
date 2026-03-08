# Cherri Language Overview

Cherri is a domain-specific language that compiles directly into Apple Shortcuts. It allows developers to author complex shortcuts using a textual syntax similar to traditional programming languages. Key goals include maintainability for large projects, transparent mapping to native Shortcut actions, and low runtime memory usage.

## Key Features
- Desktop‑friendly development with CLI, VSCode extension, and macOS IDE
- Syntax inspired by Go and Ruby with familiar constructs such as variables, conditionals, loops, and functions
- Direct one-to-one translation to underlying Shortcut actions for easy debugging
- Immutable constants and typed variables with inference
- Include system for composing large projects from multiple files
- Support for custom actions, enumerations, optionals, default values, and raw identifiers
- Copy/paste blocks for reusable action snippets
- Import questions to gather user input when shortcut is installed
- Standard library (`stdlib.cherri`) providing helpers like JavaScript execution and VCard menus

## Workflow
1. Write `.cherri` source files using the language syntax.
2. Compile with the `cherri` CLI:
   ```bash
   cherri my_shortcut.cherri
   ```
3. Optionally run with `--debug` to emit a `.plist` file and stack traces.
4. The compiler outputs a signed Shortcut ready for use on iOS or macOS.

Cherri encourages explicitness and minimal magic. Globals such as `ShortcutInput` and `CurrentDate` are constants, and most constructs map directly to Shortcuts actions to keep behavior predictable.
