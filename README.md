# Gold - The programming language

Hello, first at all this language is made for fun and research purpose, you can use it if you want.

Gold is heavily inspired by Monkey from the books of Thorsten Ball. The first version of Gold will
follow its guideline in "Writing An Interpreter In Go" and "Writing A Compiler In Go" plus some
addition of myself. I will list my modification in change logs, thus after completed these books
I will modify this language to my please, which also will be noted in the changes logs.

## Monkey base

Every change has been tested

### Lexer and tokens

Here's what I added in the lexer and tokens
- 'x++' 'x--'
- '<=' and '>='
- FLOAT are different token than INT
- FOR token

### Parser and AST

I adapted the things added in the lexer for the parser, so
- '<=' and '>='
- FLOAT are well parse
- FOR is basically the same thing than IF for now. Like if it requires parenthesis, it won't stay like that. Also 
  it's actually a while
- 'x++' and 'x--' as postfix

### Evaluator and environment

Like before I adapted the next thing in the evaluator with the following behavior
- '<=' and '>=' are classic comparator that return bool value
- FLOAT are float64 and support operation between integer in float which will convert to float64 no matter the result
- 'x++' and 'x--' return the value of the identifier then increment of decrement
- FOR loop are while loop but works

## TODO

### before cool stuff

- Reassign without 'let'
- ++x and --x
- Determine exactly when ; or \n are needed and what and every thing. In lexer I think. Doesn't seem to be needed at
  all. Should require them or add \n token and require that

### cool stuff

- real for loop
- Schrödinger cat
- tiktak
- non null value
- ? debug info
- db in runtime
