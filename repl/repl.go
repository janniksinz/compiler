package repl

import (
	"bufio"
	"fmt"
	"io"
	"monkey/compiler"
	//"monkey/evaluator"
	"monkey/lexer"
	//"monkey/object"
	"monkey/parser"
	"monkey/vm"
)

const PROMPT = ">>"

// REPL - READ EVAL PRINT LOOP
func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	//env := object.NewEnvironment()

	for {
		fmt.Fprintf(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			fmt.Fprintf(out, "Compilation failed:\n %s\n", err)
			continue
		}

		machine := vm.New(comp.Bytecode())
		err = machine.Run()
		if err != nil {
			fmt.Fprintf(out, "Executing bytecode failed:\n %s\n", err)
			continue
		}

		lastPopped := machine.LastPoppedStackElem()
		io.WriteString(out, lastPopped.Inspect())
		io.WriteString(out, "\n")

		//evaluated := evaluator.Eval(program, env)
		//if evaluated != nil {
		//	io.WriteString(out, evaluated.Inspect())
		//	io.WriteString(out, "\n")
		//}
	}
}

const PICTURE = `

	ERROR

`

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, PICTURE)
	io.WriteString(out, "Woops! We ran into some Errors here!")
	io.WriteString(out, " parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}

const _ = `
            __,__
  .--.  .-"      "-.  .--.
 / .. \/   .-. .-.  \/ .. \
| |  '|   /   Y   \  |'  | |
| \    \  \ 0 | 0 /  /   / |
 \ '-  ,\.-"""""""-./,  -' /
  ''-'  /_   ^ ^   _\  '-''
       |   \._ _./   |
       \   \ '~' /  /
        '._ '-=-' _.'
           '-----'
`
