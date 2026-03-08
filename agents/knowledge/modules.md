# Modules, Includes, and Import Questions

Cherri supports modular composition and shortcut metadata through preprocessor-style directives.

## Definitions
Use `#define` to set properties about the resulting Shortcut:
```
#define name MyShortcut
#define color red
#define glyph apple
#define inputs image, text
#define outputs app, file
#define from menubar, sleepmode, onscreen, quickactions
#define quickactions finder, services
#define noinput getclipboard
#define version 17
```
These directives correspond to Shortcut metadata such as display name, color, accepted inputs/outputs, platform availability, and default behavior when run from Quick Actions.

## Including Files
`#include 'file.cherri'` inlines another Cherri source file. Relative paths or extension-less names can be used. This enables splitting large projects into reusable modules or shared definitions such as `stdlib`.

## Import Questions
`#question variable "Prompt" "Default"` prompts the user when the shortcut is installed, storing the answer in a constant named after the variable.

## Copy and Paste
`copy name { ... }` saves a block of actions for reuse while `paste name` inserts it. This is useful for repeating sequences without introducing a custom action.

## Building and Signing
Run the compiler from the terminal:
```
cherri main.cherri
```
Use `--debug` for stack traces and a raw `.plist` output. Signing uses macOS facilities when available and falls back to a remote signing server if configured.

## Embedding and External Resources
- `embedFile(path)` base64‑encodes an asset for inclusion.
- `import` option on the CLI can convert existing Shortcuts from iCloud links.

These features allow large projects to be structured cleanly and deployed as fully signed Shortcuts.
