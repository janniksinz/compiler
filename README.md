# Implementation of the Monkey language in Golang

## Tests
Run the tests to ensure all packages are working as expected:
```bash
// Test Framework: gotest
// Test kind: Directory
go test ./... // to run tests on all packages at once
```

## Run
Start the REPL within your local go environment to test it out:
```bash
go run main.go
```
Only some main concepts of a programming language are implemented. Don't expect list comprehension here :)

## Components

- [x] Lexer
- [x] Parser
- [x] Evaluator
- [x] REPL
- [] Virtual Machine

## Features

- [x] Integers
- [x] Booleans
- [x] Prefix expressions
- [x] Infix expressions
- [x] Functions
- [x] Conditionals
- [x] Return statements
- [x] Error handling
- [x] Environment Bindings
- [x] Function calls
- [x] Strings
- [x] Builtin functions (len)
- [x] Arrays
- [x] Hashmaps

## Working on:
- [] Compiler
- [] Virtual Machine


### Compiler Structure:

- Source Code
- [x] Lexer & Parser
- AST
- [] Optimizer
- Internal Representation
- [] Code Generator
- Machine Code
