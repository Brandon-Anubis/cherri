# Actions and Custom Functions

Actions are the core building blocks of Cherri programs. Each action corresponds to an Apple Shortcut action and accepts typed parameters.

## Calling Actions
Invoke actions like functions:
```
alert("Hello", "Title")
resizeImage(image, "640", "480")
```
Parameters are passed positionally. Use string interpolation or variables to supply dynamic values.

## Standard Actions
Cherri ships with thousands of predefined actions organized in the `actions/` directory. Each action is defined using `#define action` and maps to a Shortcut identifier and its parameter keys. For example:
```
#define action 'exit' stop()
#define action comment(rawtext text: 'WFCommentActionText')
```
You may call `stop()` or `comment('Some text')` directly after these definitions are loaded (automatically by the compiler).

## Custom Actions
Define new actions using the `action` keyword:
```
action add(number op1, number op2): number {
    const result = op1 + op2
    output("{result}")
}
```
- Parameters specify type and name.
- Return type follows a colon after the parameter list.
- Use `output(value)` to return a result.
- Recursion is supported as demonstrated in a `fibonacci` example.

## Mapping to Existing Shortcuts
`#define action` can expose existing Shortcut actions with custom parameter names and defaults:
```
#define action 'dropbox.savefile' saveToDropbox(
    variable file: 'WFInput',
    text path: 'WFFileDestinationPath',
    bool ?overwrite: 'WFSaveFileOverwrite' = false
) {
    "WFAskWhereToSave": false
}
```
The block may include additional key–value pairs that are always set when the action is used.

## Raw Actions
For actions not yet defined, use `rawAction` with an identifier and parameter dictionary:
```
rawAction("is.workflow.actions.alert", {
    "WFAlertActionMessage": "Hello, world!",
    "WFAlertActionTitle": "Alert"
})
```

## Copy/Paste Blocks
Reuse sequences of actions with macros:
```
copy snippet {
    alert("Hello")
}

paste snippet
paste snippet
```
Each `paste` inserts the copied actions at that location.

## Output Management
- Most actions output a value that becomes the implicit input for the next action.
- Use `output(value)` in custom actions to set the action result.
- Access `RepeatIndex` and other special variables inside loops to use intermediate results.
