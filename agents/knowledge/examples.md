# Example Snippets

## Hello World
```
/* Hello, World! */
#define name Hello, Cherri!
@message = "Hello, Cherri!"
@title = "Alert"
alert(message, "{title}")
```

## Loops and Conditionals
```
@count = 3
repeat i for 6 {
    if count == i {
        alert("Reached {i}")
    }
}

@items = list("a","b","c")
for item in items {
    alert(RepeatIndex,item)
}
```

## Menu
```
menu "Choose" {
    item "One":
        alert("1")
    item "Two":
        alert("2")
}
```

## Custom Action
```
action add(number a, number b): number {
    const result = a + b
    output("{result}")
}
@sum = add(2,2)
show("{sum}")
```

## Import Question and Include
```
#question name "What is your name?" "Brandon"
#include 'stdlib'
alert(name, "User")
```
