package compiler

import (
	"errors"
	"fmt"
	"gold/ast"
	"gold/code"
	"gold/object"
	"sort"
)

type Compiler struct {
	constants []object.Object

	symbolTable *SymbolTable

	scopes     []CompilationScope
	scopeIndex int
}

func New() *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{}, // NOTE : to replace lastInstruction when we pop it, but is it really usefull ?
	}

	symbolTable := NewSymbolTable()

	for i, v := range object.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	return &Compiler{
		constants:   []object.Object{},
		symbolTable: symbolTable,
		scopes:      []CompilationScope{mainScope},
		scopeIndex:  0,
	}
}

func NewWithState(s *SymbolTable, constants []object.Object) *Compiler {
	compiler := New()
	compiler.symbolTable = s
	compiler.constants = constants
	return compiler
}

// Compile : create the bytecode from the instructions in the AST and add it in the compiled instructions.
// When it encounters an Integer or a function, add it on the pool of constant. To query it
// we use the index in the constant pool and the op opConstant.
func (c *Compiler) Compile(node ast.Node) (error, map[object.ObjectType]bool) {
	var err error
	objectTypeSet := make(map[object.ObjectType]bool)
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			err, objectTypeSet = c.Compile(s)
			if err != nil {
				return err, objectTypeSet
			}
		}

	case *ast.ExpressionStatement:
		err, objectTypeSet = c.Compile(node.Expression)
		if err != nil {
			return err, objectTypeSet
		}
		c.emit(code.OpPop)

	case *ast.InfixExpression:
		// This separate case reverse the order of right and left. With that we can use the same opCode for < and >
		// TODO : check type everywhere

		if node.Operator == "<" || node.Operator == "<=" {
			err, rightObjectTypeSet := c.Compile(node.Right)
			if err != nil {
				return err, objectTypeSet
			}
			for k, v := range rightObjectTypeSet {
				objectTypeSet[k] = v
			}

			err, leftObjectTypeSet := c.Compile(node.Left)
			if err != nil {
				return err, objectTypeSet
			}
			for k, v := range leftObjectTypeSet {
				objectTypeSet[k] = v
			}

			if node.Operator == "<" {
				c.emit(code.OpGreaterThan)
			} else {
				c.emit(code.OpGreaterEqualThan)
			}
			return nil, objectTypeSet
		}

		err, leftObjectTypeSet := c.Compile(node.Left)
		if err != nil {
			return err, objectTypeSet
		}
		for k, v := range leftObjectTypeSet {
			objectTypeSet[k] = v
		}

		err, rightObjectTypeSet := c.Compile(node.Right)
		if err != nil {
			return err, objectTypeSet
		}
		for k, v := range rightObjectTypeSet {
			objectTypeSet[k] = v
		}

		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case ">":
			c.emit(code.OpGreaterThan)
		case ">=":
			c.emit(code.OpGreaterEqualThan)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator), objectTypeSet
		}

	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))

	case *ast.FloatLiteral:
		float := &object.Float{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(float))

	case *ast.IncPostExpression:
		// Compile twice to have two OpGet to still have one after modification
		symbol, ok := c.symbolTable.Resolve(node.Left.Value)
		if !ok {
			return nil, objectTypeSet
		}
		if symbol.Scope == GlobalScope {
			c.emit(code.OpGetGlobal, symbol.Index)
			c.emit(code.OpGetGlobal, symbol.Index)
		} else {
			c.emit(code.OpGetLocal, symbol.Index)
			c.emit(code.OpGetLocal, symbol.Index)
		}

		switch node.Operator {
		case "++":
			c.emit(code.OpInc)
		case "--":
			c.emit(code.OpDec)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator), objectTypeSet
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.IncPreExpression:
		// NOTE : could need less code and let OpInc do everything to limit the bytecode
		// The problem is to choose global or local

		// Compile twice to have two OpGet to still have one after modification
		symbol, ok := c.symbolTable.Resolve(node.Right.Value)
		if !ok {
			return fmt.Errorf("unknown name %s", node.Right.Value), objectTypeSet
		}
		if symbol.Scope == GlobalScope {
			c.emit(code.OpGetGlobal, symbol.Index)
		} else {
			c.emit(code.OpGetLocal, symbol.Index)
		}

		switch node.Operator {
		case "++":
			c.emit(code.OpInc)
		case "--":
			c.emit(code.OpDec)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator), objectTypeSet
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpGetGlobal, symbol.Index)
		} else {
			c.emit(code.OpGetLocal, symbol.Index)
		}

	case *ast.Boolean:
		if node.Value { // True and False aren't in constant pool, there are separate object in VM
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}

	case *ast.Null:
		c.emit(code.OpNull)
		objectTypeSet[object.NULL_OBJ] = true

	case *ast.PrefixExpression:
		err, objectTypeSet = c.Compile(node.Right)
		if err != nil {
			return err, objectTypeSet
		}

		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator), objectTypeSet
		}

	case *ast.IfExpression:
		err, _ = c.Compile(node.Condition)
		if err != nil {
			return err, objectTypeSet
		}

		// Emit an `OpJumpNotTruthy` with a bogus value
		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err, objectTypeSet = c.Compile(node.Consequence)
		if err != nil {
			return err, objectTypeSet
		}

		if c.lastInstructionIs(code.OpPop) {
			c.removeLastPop()
		} else {
			c.emit(code.OpNull)
		}

		// Emit an `OpJump` with a bogus value
		jumpPos := c.emit(code.OpJump, 9999)

		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		if node.Alternative == nil {
			c.emit(code.OpNull)
			objectTypeSet[object.NULL_OBJ] = true
		} else {
			err, altObjectTypeSet := c.Compile(node.Alternative)
			for k, v := range altObjectTypeSet {
				objectTypeSet[k] = v
			}
			if err != nil {
				return err, objectTypeSet
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			} else {
				c.emit(code.OpNull)
			}
		}

		afterAlternativePos := len(c.currentInstructions())
		c.changeOperand(jumpPos, afterAlternativePos)

	case *ast.WhileExpression:
		pos := len(c.currentInstructions())

		err, _ = c.Compile(node.Condition)
		if err != nil {
			return err, objectTypeSet
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err, _ = c.Compile(node.Consequence)
		if err != nil {
			return err, objectTypeSet
		}

		c.emit(code.OpJump, pos)

		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		c.emit(code.OpNull) // NOTE : since it's an expression, must produce a value
		objectTypeSet[object.NULL_OBJ] = true

	case *ast.BlockStatement:
		for _, s := range node.Statements {
			err, tmpobjectTypeSet := c.Compile(s)
			for k, v := range tmpobjectTypeSet {
				objectTypeSet[k] = v
			}
			if err != nil {
				return err, objectTypeSet
			}
		}

	case *ast.LetStatement:
		symbol := c.symbolTable.Define(node.Name.Value, false)
		err, objectTypeSet = c.Compile(node.Value)
		if err != nil {
			return err, objectTypeSet
		}

		if _, ok := objectTypeSet[object.NULL_OBJ]; ok {
			return errors.New("can't use 'let' statement with null"), objectTypeSet
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.MayStatement:
		// TODO : again very similar to let, can refactor
		symbol := c.symbolTable.Define(node.Name.Value, true)
		err, _ = c.Compile(node.Value)
		if err != nil {
			return err, objectTypeSet
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.ReassignStatement:
		symbol, ok := c.symbolTable.Resolve(node.Name.Value)
		if !ok {
			return errors.New("undefined variable " + node.Name.Value), objectTypeSet
		}
		err, objectTypeSet = c.Compile(node.Value)
		if err != nil {
			return err, objectTypeSet
		}

		var canReturnNull bool
		if _, canReturnNull = objectTypeSet[object.NULL_OBJ]; !symbol.Nullable && canReturnNull {
			return errors.New("your value is not nullable"), objectTypeSet
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if !ok {
			return fmt.Errorf("undefined variable %s", node.Value), objectTypeSet
		}

		c.loadSymbol(symbol)
		if symbol.Nullable {
			objectTypeSet[object.NULL_OBJ] = true
		}

	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))

	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			err, _ = c.Compile(el)
			if err != nil {
				return err, objectTypeSet
			}
		}

		c.emit(code.OpArray, len(node.Elements))

	case *ast.HashLiteral:
		keys := []ast.Expression{}
		for k := range node.Pairs {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})

		for _, k := range keys {
			// TODO : check nullable
			err, _ = c.Compile(k)
			if err != nil {
				return err, objectTypeSet
			}
			err, _ = c.Compile(node.Pairs[k])
			if err != nil {
				return err, objectTypeSet
			}
		}

		c.emit(code.OpHash, len(node.Pairs)*2)

	case *ast.IndexExpression:
		// TODO : error if null ?
		err, objectTypeSet = c.Compile(node.Left)
		if err != nil {
			return err, objectTypeSet
		}

		err, objectTypeSet = c.Compile(node.Index)
		if err != nil {
			return err, objectTypeSet
		}

		c.emit(code.OpIndex)

	case *ast.FunctionLiteral:
		c.enterScope()

		if node.Name != "" {
			c.symbolTable.DefineFunctionName(node.Name)
		}

		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value, true)
		}

		err, objectTypeSet = c.Compile(node.Body)
		if err != nil {
			return err, objectTypeSet
		}

		if !c.lastInstructionIs(code.OpReturn) {
			c.emit(code.OpNull)
			c.emit(code.OpReturn)
		}

		freeSymbols := c.symbolTable.FreeSymbols
		numLocals := c.symbolTable.numDefinitions
		instructions := c.leaveScope()

		for _, s := range freeSymbols {
			c.loadSymbol(s)
		}

		compiledFn := &object.CompiledFunction{
			Instructions:  instructions,
			NumLocals:     numLocals,
			NumParameters: len(node.Parameters),
		}

		fnIndex := c.addConstant(compiledFn)
		c.emit(code.OpClosure, fnIndex, len(freeSymbols))

	case *ast.ReturnStatement:
		err, objectTypeSet = c.Compile(node.ReturnValue)
		if err != nil {
			return err, objectTypeSet
		}

		c.emit(code.OpReturn)

	case *ast.CallExpression:
		err, objectTypeSet = c.Compile(node.Function)
		if err != nil {
			return err, objectTypeSet
		}

		for _, a := range node.Arguments {
			err, _ = c.Compile(a)
			if err != nil {
				return err, objectTypeSet
			}
		}

		c.emit(code.OpCall, len(node.Arguments))

	}

	return err, objectTypeSet
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.currentInstructions(),
		Constants:    c.constants,
	}
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)

	c.setLastInstruction(op, pos)

	return pos
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.currentInstructions())
	updatedInstructions := append(c.currentInstructions(), ins...)

	c.scopes[c.scopeIndex].instructions = updatedInstructions

	return posNewInstruction
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}

	c.scopes[c.scopeIndex].previousInstruction = previous
	c.scopes[c.scopeIndex].lastInstruction = last
}

func (c *Compiler) lastInstructionIs(op code.Opcode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}

	return c.scopes[c.scopeIndex].lastInstruction.Opcode == op
}

func (c *Compiler) removeLastPop() {
	last := c.scopes[c.scopeIndex].lastInstruction
	previous := c.scopes[c.scopeIndex].previousInstruction

	old := c.currentInstructions()
	new := old[:last.Position]

	c.scopes[c.scopeIndex].instructions = new
	c.scopes[c.scopeIndex].lastInstruction = previous
}

func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	ins := c.currentInstructions()

	for i := 0; i < len(newInstruction); i++ {
		ins[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.currentInstructions()[opPos])
	newInstruction := code.Make(op, operand)

	c.replaceInstruction(opPos, newInstruction)
}

func (c *Compiler) currentInstructions() code.Instructions {
	return c.scopes[c.scopeIndex].instructions
}

func (c *Compiler) enterScope() {
	scope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++

	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveScope() code.Instructions {
	instructions := c.currentInstructions()

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--

	c.symbolTable = c.symbolTable.Outer

	return instructions
}

func (c *Compiler) loadSymbol(s Symbol) {
	switch s.Scope {
	case GlobalScope:
		c.emit(code.OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(code.OpGetLocal, s.Index)
	case BuiltinScope:
		c.emit(code.OpGetBuiltin, s.Index)
	case FreeScope:
		c.emit(code.OpGetFree, s.Index)
	case FunctionScope:
		c.emit(code.OpCurrentClosure)
	}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

type CompilationScope struct {
	instructions        code.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}
