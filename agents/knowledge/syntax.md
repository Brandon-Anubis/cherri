# Cherri Syntax and Grammar

Cherri uses a concise syntax with significant whitespace only inside strings. Statements generally end at newlines. Comments may be written with `//` for single lines or `/* ... */` for blocks.

## Variables and Constants
- Variables start with `@` and can be assigned using `=`: `@name = "Brandon"`.
- Immutable constants use the `const` keyword: `const greeting = "Hello"`.
- Variables may declare a type with `:@type`: `@count: number`.
- Strings support interpolation with `{expression}` inside double quotes.
- Single quotes define raw strings that forbid interpolation and newlines but compile faster.

## Collections
- Arrays use `[item1, item2]` syntax and can mix types.
- Dictionaries mirror JSON object syntax:
  ```
  @dict = {
      "key": "value",
      "nested": [1,2,3]
  }
  ```
- Access dictionary values with `getValue(dict,"key")` or bracket syntax `dict['key']` (no interpolation).

## Enumerations
Declare enums with `enum`:
```
enum callType {
    'Audio',
    'Video'
}
```
Enum members are quoted strings.

## Expressions and Operators
Cherri supports standard arithmetic and logical operators:
`+`, `-`, `*`, `/`, `%`, `==`, `!=`, `>`, `<`, `>=`, `<=`, `&&`, `||`, `contains`, `!contains`, `beginsWith`, `endsWith`, and range `<>` for between checks.
Compound assignments `+=`, `-=`, `*=`, `/=` are also available.

## Control Flow
### Conditionals
```
if condition {
    // actions
} else {
    // alternative
}
```
Multiple conditions can be combined with `&&` and `||`. Blocks close with `}` and optional `else`.

### Loops
- Repeat a fixed number of times: `repeat i for 6 { ... }`
- Iterate over a collection: `for item in listVar { ... }`
The special variable `RepeatIndex` provides the current iteration index inside `for` loops.

### Menus
Create interactive menus:
```
menu "Prompt" {
    item "Option 1":
        alert("You chose option 1")
    item "Option 2":
        alert("You chose option 2")
}
```

## String and Type Utilities
- Convert types using suffixes: `number.text` coerces a number to text.
- Global constants like `ShortcutInput`, `CurrentDate`, and `Device` can be referenced directly or interpolated into strings.
- `nil` represents an absent value and may be supplied to actions, loops, or comparisons.

## Macros
- `copy name { ... }` defines a reusable block of actions.
- `paste name` inserts the copied block at that location.

## File Structure
Top-level directives begin with `#`:
- `#define` sets metadata such as `color`, `glyph`, `name`, etc.
- `#include 'file.cherri'` imports another Cherri file.
- `#question var "Prompt" "Default"` asks for input when the shortcut is imported.
