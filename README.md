# Gold - The Programming Language (Experimental & Research Purpose)

<img src="assets/mascott.jpg" alt="CamCam the cameleon (not definitive)" width="300">

Hello, Gold is currently under development and not intended for production use. It's primarily for learning and experimentation purposes.

## Language Overview:

- Compiled language with a virtual machine
- Object-oriented tendencies (work in progress)
- Type safety with null protection
- Recent strong focus on catching errors at compile time to minimize runtime issues

With this repo, you can compile your code in Gold, run it in a VM and even use a REPL.

### Inspiration:
Gold draws from Monkey, the language featured in Thorsten Ball's books "Writing An Interpreter In Go" and "Writing A Compiler In Go". While initially following its guidelines, Gold has evolved into a distinct language.

### More Information:
A basic wiki is available below to guide you through Gold's core concepts. A more comprehensive wiki will be created when the language matures and resources permit.


## Feature

### Basics:

- Standard arithmetic operators (+, -, /, *, ==, !=, >, <, <=, >=, !)
- Prefix and postfix increment/decrement (++, --)
- Primitive types: int, float, bool, string, array, dictionary
- Type-based error checking during compilation
- Automatic type conversions (e.g., int + float) when possible
- Array and dictionary behavior similar to Python (can hold any type as keys or values)

Here are some examples:

```
let x = 0
while (x++ < 10) {
  print(x) // print all numbers from 1 to 10
}

[[1, 1, 1]][0][0] // will produce 1
```

There are also some built-in functions with obvious behavior : 
- *print*
- *len*
- *push*
- *first*
- *last*

### Typed Variables:

The incorporation of typed properties and null safety is a pivotal aspect of the language, and I invested considerable effort in refining it during the development process. Here's how it works :

- Explicit type declaration using keywords like *mint*, *lint*, *mstr*, *lstr*, *mflt*, *lflt*, *marr*, *larr*, *mdct*, *ldct*, *any*, *may*, and *let*.
- *m* or *l* prefix indicates whether the value can be null (*may*) or must be non-null (*let*).
- Use *let* or *may* without types to let the compiler infer the type.

The most important part of it comes with functions.

### Typed function

The functions work like variables with small nuance. You will declare them the same way you declare a variable but the type is actually the type returned by the function. This is particularly useful for high-order functions or closures.


** This will work
```
lint x = 0
mint f = fn(mint a) {
  if (5 > 2) {
    return a
  }
}
f(x)
```
** This will fail
```
lint x = 0
mint f = fn(mstr a) {
  if (5 > 2) {
    return a
  }
}
f(x)
```

### Everything Is an Expression (Work in Progress):

*if* and *while* statements can potentially return values like functions (experimental feature).

## Known Issues:

*len* function: The compiler accepts any type, but the VM will catch errors.
*++x--*: Unsupported (and unnecessary, who wants to do that ?).

## Usage 
Given that this language is built on Go, you can easily initiate the REPL by running `go run main.go`. To compile a file named test.gold, use the command `go run main.go` compile test. This will generate a file called test.cold, which you can execute with `go run main.go run test`. Alternatively, you can simplify the language installation using go install (ensure that you add GOPATH to your PATH).