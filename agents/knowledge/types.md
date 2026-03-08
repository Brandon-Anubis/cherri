# Data Types and Conversions

Cherri offers a rich type system with inference and explicit annotations.

## Primitive Types
- **text**: UTF‑8 strings. Interpolated with `{}` when in double quotes.
- **rawtext**: Single‑quoted strings without interpolation or newlines.
- **number**: Integer values.
- **float**: Decimal numbers.
- **bool**: `true` or `false`.
- **date**: Date objects created with `date("October 5, 2022")` or derived from globals.
- **nil**: Absence of value; may be passed to actions or compared directly.

## Collections
- **array**: Ordered list `[1, "two", true]`. Use `list("a","b")` to create from values.
- **dictionary**: JSON-style map `{ "key": value }` supporting nested structures and arrays.

## Type Annotations and Inference
Variables can optionally declare a type: `@value: number`. When assigned, the compiler validates the type but also infers it from expressions.

## Globals and Special Types
Certain identifiers are predefined:
- `ShortcutInput`, `CurrentDate`, `Clipboard`, `Device`, `Ask`, etc.
- Globals are case-sensitive and behave like constants.

## Type Casting
- Use property-style accessors: `variable.text`, `variable.number`, etc.
- `getAs` syntax retrieves values from dictionaries or globals: `Device['System Version'].text`.
- In strings, casting can occur inline: `"Version {Device['OS']}"`.

## Enumerations
Enums constrain values to a predefined set:
```
enum callType {
    'Audio',
    'Video'
}
```
Enum names can be used as parameter types for actions.

## Optional Types
Prefix a parameter with `?` to mark it optional in action definitions: `bool ?overwrite = false`.

## Default Values
Provide `= value` after a parameter or constant to supply a default.
