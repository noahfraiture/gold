package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"gold/compiler"
	"gold/lexer"
	"gold/object"
	"gold/parser"
	"gold/repl"
	"gold/vm"
	"os"
	"os/user"
)

func main() {
	args := os.Args

	switch len(args) {
	case 1:
		user, err := user.Current()
		if err != nil {
			panic(err)
		}
		fmt.Printf("Hello %s! This is the Gold programming language!\n",
			user.Username)
		fmt.Printf("Feel free to type in commands\n")
		repl.Start(os.Stdin, os.Stdout)
		return

	case 3:
		var err error
		switch args[1] {
		case "compile", "c":
			err = compileFile(args[2]+".gold", args[2]+".cold")
		case "vm", "v", "run", "r":
			err = runBinaryFile(args[2] + ".cold")
		default:
			panic("unknown command")
		}
		if err != nil {
			panic(err)
		}
	}
}

func compileFile(inputFileName, outputFileName string) error {
	inputFile, err := os.ReadFile(inputFileName)
	if err != nil {
		panic(err)
	}

	l := lexer.New(string(inputFile))
	p := parser.New(l)
	program := p.ParseProgram()
	comp := compiler.New()
	_, err = comp.Compile(program)
	if err != nil {
		return err
	}

	bytesBuffer, err := writeBytecode(comp.Bytecode())
	if err != nil {
		return err
	}

	outputFile, err := os.OpenFile(outputFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = outputFile.Write(bytesBuffer.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func writeBytecode(bytecode *compiler.Bytecode) (*bytes.Buffer, error) {
	register()
	var bytesBuffer bytes.Buffer
	enc := gob.NewEncoder(&bytesBuffer)

	err := enc.Encode(bytecode)
	if err != nil {
		return nil, err
	}
	return &bytesBuffer, nil
}

func register() {
	l := []object.Object{
		&object.Integer{},
		&object.Float{},
		&object.Boolean{},
		&object.Null{},
		&object.ReturnValue{},
		&object.Function{},
		&object.String{},
		&object.Builtin{},
		&object.Array{},
		&object.Hash{},
		&object.CompiledFunction{},
		&object.Closure{},
	}

	for _, t := range l {
		gob.Register(t)
	}
}

func runBinaryFile(fileName string) error {
	bytecode, err := readBytecode(fileName)
	if err != nil {
		return err
	}

	vm := vm.New(bytecode)
	err = vm.Run()
	if err != nil {
		return err
	}
	return nil
}

func readBytecode(filename string) (*compiler.Bytecode, error) {
	var bytesBuffer bytes.Buffer
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	_, err = bytesBuffer.ReadFrom(file)
	if err != nil {
		return nil, err
	}

	register()
	dec := gob.NewDecoder(&bytesBuffer)
	var bytecode compiler.Bytecode
	err = dec.Decode(&bytecode)
	if err != nil {
		return nil, err
	}

	return &bytecode, nil
}
