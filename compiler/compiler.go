package compiler

import (
	"fmt"
	"gold/ast"
	"gold/code"
	"gold/object"
	"sort"
)

type Compiler struct {
	constants   []object.Object
	symbolTable *SymbolTable
	scopes      []CompilationScope
	scopeIndex  int
}

func New() *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{}, // NOTE : to replace lastInstruction when we pop it, but is it really usefull ?
	}

	symbolTable := NewSymbolTable()

	for i, v := range object.Builtins {
		symbolTable.DefineBuiltin(i, v.Name, v.Type)
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
func (c *Compiler) Compile(node ast.Node) (object.Attribute, error) {
	var err error
	infos := object.Attribute{}
	switch node := node.(type) {

	// === MAIN ===
	case *ast.Program:
		for _, s := range node.Statements {
			infos, err = c.Compile(s)
			if err != nil {
				return infos, err
			}
		}

	case *ast.ExpressionStatement:
		infos, err = c.Compile(node.Expression)
		if err != nil {
			return infos, err
		}
		c.emit(code.OpPop)

	case *ast.BlockStatement:
		for _, s := range node.Statements {
			tmpobjectTypeSet, err := c.Compile(s)
			if err != nil {
				return infos, err
			}

			if infos.ObjectType == "" {
				infos.ObjectType = tmpobjectTypeSet.ObjectType
				continue
			}

			if tmpobjectTypeSet.ObjectType == "" {
				continue
			}

			if !infos.IsTypeOf(tmpobjectTypeSet.ObjectType) {
				return infos, fmt.Errorf("block statement can return different types. old=%s current=%s",
					infos.ObjectType, tmpobjectTypeSet.ObjectType)
			}
		}

	// === EXPRESSION ===
	case *ast.IfExpression:
		// Here we don't check the condition type to accept every truthy type
		_, err := c.Compile(node.Condition)
		if err != nil {
			return infos, err
		}

		// Emit an `OpJumpNotTruthy` with a bogus value
		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		infos, err = c.Compile(node.Consequence)
		if err != nil {
			return infos, err
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
			infos.Nullable = true
		} else {
			altObjectTypeSet, err := c.Compile(node.Alternative)
			infos.Nullable = altObjectTypeSet.Nullable || infos.Nullable
			if infos.ObjectType == object.NULL_OBJ {
				infos.ObjectType = altObjectTypeSet.ObjectType
			}

			bothExist := altObjectTypeSet.ObjectType != "" && infos.ObjectType != ""
			sameType := infos.IsTypeOf(altObjectTypeSet.ObjectType)

			if bothExist && !sameType {
				return infos, fmt.Errorf("consquence=%s must be same type of alternative=%s", infos.ObjectType, altObjectTypeSet.ObjectType)
			}
			if err != nil {
				return infos, err
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

		// Here we don't check the condition type to accept every truthy type
		_, err := c.Compile(node.Condition)
		if err != nil {
			return infos, err
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		_, err = c.Compile(node.Consequence) // NOTE : will have to get the infos when while return value. How to ignore return and only take break return value?
		if err != nil {
			return infos, err
		}

		c.emit(code.OpJump, pos)

		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		c.emit(code.OpNull) // NOTE : since it's an expression, must produce a value
		infos.Nullable = true

	case *ast.InfixExpression:
		// This separate case reverse the order of right and left. With that we can use the same opCode for < and >

		var rightInfos object.Attribute
		var leftInfos object.Attribute
		var err error

		if node.Operator == "<" || node.Operator == "<=" {

			leftInfos, err = c.Compile(node.Right)
			if err != nil {
				return infos, err
			}

			rightInfos, err = c.Compile(node.Left)
			if err != nil {
				return infos, err
			}

		} else {
			leftInfos, err = c.Compile(node.Left)
			if err != nil {
				return infos, err
			}

			rightInfos, err = c.Compile(node.Right)
			if err != nil {
				return infos, err
			}
		}

		bothInteger := rightInfos.IsTypeOf(object.INTEGER_OBJ) && leftInfos.IsTypeOf(object.INTEGER_OBJ)
		bothNumber := rightInfos.IsTypeOf(object.INTEGER_OBJ, object.FLOAT_OBJ) && leftInfos.IsTypeOf(object.INTEGER_OBJ, object.FLOAT_OBJ)
		bothString := rightInfos.IsTypeOf(object.STRING_OBJ) && leftInfos.IsTypeOf(object.STRING_OBJ)

		switch node.Operator {
		case "+", "-", "*", "/":
			if bothInteger {
				infos.ObjectType = object.INTEGER_OBJ
			} else if bothNumber {
				infos.ObjectType = object.FLOAT_OBJ
			} else if bothString {
				infos.ObjectType = object.STRING_OBJ
			}
		case ">", "<", ">=", "<=", "==", "!=":
			infos.ObjectType = object.BOOLEAN_OBJ
		}

		switch node.Operator {
		case "+":
			if !bothString && !bothNumber {
				return infos, fmt.Errorf("trying to do '%s' with other than numbers or string. left=%s right=%s",
					node.Operator, leftInfos.ObjectType, rightInfos.ObjectType)
			}
			c.emit(code.OpAdd)
		case "-":
			if !bothNumber {
				return infos, fmt.Errorf("trying to do '%s' with other than numbers. left=%s right=%s",
					node.Operator, leftInfos.ObjectType, rightInfos.ObjectType)
			}
			c.emit(code.OpSub)
		case "*":
			if !bothNumber {
				return infos, fmt.Errorf("trying to do '%s' with other than numbers. left=%v right=%v",
					node.Operator, leftInfos.ObjectType, rightInfos.ObjectType)
			}
			c.emit(code.OpMul)
		case "/":
			if !bothNumber {
				return infos, fmt.Errorf("trying to do '%s' with other than numbers. left=%s right=%s",
					node.Operator, leftInfos.ObjectType, rightInfos.ObjectType)
			}
			c.emit(code.OpDiv)
		case ">", "<":
			if !bothString && !bothNumber {
				return infos, fmt.Errorf("trying to do '%s' with other than numbers or string. left=%s right=%s",
					node.Operator, leftInfos.ObjectType, rightInfos.ObjectType)
			}
			c.emit(code.OpGreaterThan)
		case ">=", "<=":
			if !bothString && !bothNumber {
				return infos, fmt.Errorf("trying to do '%s' with other than numbers or string. left=%s right=%s",
					node.Operator, leftInfos.ObjectType, rightInfos.ObjectType)
			}
			c.emit(code.OpGreaterEqualThan)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		default:
			return infos, fmt.Errorf("unknown operator '%s'", node.Operator)
		}

		// NOTE : is a new structure, is it needed ? could reuse prefix and create postfilx

	case *ast.IncPostExpression:
		// Compile twice to have two OpGet to still have one after modification
		symbol, ok := c.symbolTable.Resolve(node.Left.Value)
		if !ok {
			return infos, nil
		}

		if !symbol.ObjectInfo.IsTypeOf(object.INTEGER_OBJ, object.FLOAT_OBJ) {
			return infos, fmt.Errorf("trying to do '%s' on other than numbers. symbol=%s", node.Operator, symbol.ObjectInfo.ObjectType)
		}
		infos = symbol.ObjectInfo

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
			return infos, fmt.Errorf("unknown operator %s", node.Operator)
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
			return infos, fmt.Errorf("unknown name %s", node.Right.Value)
		}

		if !symbol.ObjectInfo.IsTypeOf(object.INTEGER_OBJ, object.FLOAT_OBJ) {
			return infos, fmt.Errorf("trying to do '%s' on other than numbers. symbol=%s", node.Operator, symbol.ObjectInfo.ObjectType)
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
			return infos, fmt.Errorf("unknown operator %s", node.Operator)
		}
		infos = symbol.ObjectInfo

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

	case *ast.PrefixExpression:
		infos, err = c.Compile(node.Right)
		if err != nil {
			return infos, err
		}

		switch node.Operator {
		case "!":
			// NOTE : every value can be truthy, so we don't check type
			c.emit(code.OpBang)
		case "-":
			if !infos.IsTypeOf(object.INTEGER_OBJ, object.FLOAT_OBJ) {
				return infos, fmt.Errorf("trying to do '%s' on other than numbers", node.Operator)
			}
			c.emit(code.OpMinus)
		default:
			return infos, fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.IndexExpression:
		// NOTE : since array and map accept anything, it's impossible to define a  type
		implemInfos, err := c.Compile(node.Left)
		if err != nil {
			return infos, err
		}
		if !implemInfos.IsTypeOf(object.ARRAY_OBJ, object.HASH_OBJ) {
			return infos, fmt.Errorf("trying to index something other than array or hash")
		}

		indexInfos, err := c.Compile(node.Index)
		if err != nil {
			return infos, err
		}
		if indexInfos.ObjectType != object.INTEGER_OBJ {
			return infos, fmt.Errorf("trying to index with non integer")
		}

		c.emit(code.OpIndex)
		infos.ObjectType = object.UNDEFINED

	// === VALUE ===
	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))
		infos.ObjectType = object.INTEGER_OBJ

	case *ast.FloatLiteral:
		float := &object.Float{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(float))
		infos.ObjectType = object.FLOAT_OBJ

	case *ast.Boolean:
		if node.Value { // True and False aren't in constant pool, there are separate object in VM
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
		infos.ObjectType = object.BOOLEAN_OBJ

	case *ast.Null:
		c.emit(code.OpNull)
		infos.Nullable = true
		infos.ObjectType = object.NULL_OBJ

	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))
		infos.ObjectType = object.STRING_OBJ

	case *ast.ArrayLiteral:
		// NOTE : currently allow nullable value without any trouble
		for _, el := range node.Elements {
			_, err = c.Compile(el)
			if err != nil {
				return infos, err
			}
		}

		c.emit(code.OpArray, len(node.Elements))
		infos.ObjectType = object.ARRAY_OBJ

	case *ast.HashLiteral:
		// NOTE : currently allow nullable value without any trouble
		keys := []ast.Expression{}
		for k := range node.Pairs {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})

		for _, k := range keys {
			// TODO : check nullable
			_, err = c.Compile(k)
			if err != nil {
				return infos, err
			}
			_, err = c.Compile(node.Pairs[k])
			if err != nil {
				return infos, err
			}
		}

		c.emit(code.OpHash, len(node.Pairs)*2)
		infos.ObjectType = object.HASH_OBJ

	// === DECLARE ===
	case *ast.LetDeclare:
		infos, err = c.Compile(node.Value)
		if err != nil {
			return infos, err
		}

		if infos.Nullable && !node.Nullable {
			return infos, errorNullable(node.Name.Value)
		}

		symbol := c.symbolTable.Define(node.Name.Value, object.Attribute{Nullable: node.Nullable, ObjectType: infos.ObjectType})

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.AnyDeclare:
		err := c.compileDeclare(node.Name.Value, node.Nullable, node.Value, object.UNDEFINED) // Will be nullable
		if err != nil {
			return infos, err
		}

	case *ast.IntDeclare:
		err := c.compileDeclare(node.Name.Value, node.Nullable, node.Value, object.INTEGER_OBJ)
		if err != nil {
			return infos, err
		}

	case *ast.FloatDeclare:
		err := c.compileDeclare(node.Name.Value, node.Nullable, node.Value, object.FLOAT_OBJ)
		if err != nil {
			return infos, err
		}

	case *ast.StrDeclare:
		err := c.compileDeclare(node.Name.Value, node.Nullable, node.Value, object.STRING_OBJ)
		if err != nil {
			return infos, err
		}

	case *ast.ArrDeclare:
		err := c.compileDeclare(node.Name.Value, node.Nullable, node.Value, object.ARRAY_OBJ)
		if err != nil {
			return infos, err
		}

	case *ast.DctDeclare:
		err := c.compileDeclare(node.Name.Value, node.Nullable, node.Value, object.HASH_OBJ)
		if err != nil {
			return infos, err
		}

	case *ast.ReassignStatement:
		symbol, ok := c.symbolTable.Resolve(node.Name.Value)
		if !ok {
			return infos, errorUndefined(node.Name.Value)
		}
		infos, err = c.Compile(node.Value)
		if err != nil {
			return infos, err
		}

		if !symbol.ObjectInfo.Nullable && infos.Nullable {
			return infos, errorNullable(symbol.Name)
		}

		if !infos.IsTypeOf(symbol.ObjectInfo.ObjectType) {
			return infos, errorType(symbol.Name, symbol.ObjectInfo.ObjectType, infos.ObjectType)
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	// === IDENTIFIER ===

	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if !ok {
			return infos, fmt.Errorf("undefined variable %s", node.Value)
		}

		c.loadSymbol(symbol)
		infos = symbol.ObjectInfo

	case *ast.FunctionLiteral:
		c.enterScope()

		if node.Name != "" {
			// UNDEFINED since the function itself has no type but the variabe associated has.
			// But if we decide to modify the definition to include a type, it can be add there.
			c.symbolTable.DefineFunctionName(node.Name, object.Attribute{ObjectType: object.UNDEFINED, Nullable: false})
		}

		// TODO : function accept arguments of any type and function
		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value, object.Attribute{ObjectType: object.UNDEFINED, Nullable: true})
		}

		infos, err = c.Compile(node.Body)
		if err != nil {
			return infos, err
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
		infos, err = c.Compile(node.ReturnValue)
		if err != nil {
			return infos, err
		}

		c.emit(code.OpReturn)

	case *ast.CallExpression:
		infos, err = c.Compile(node.Function)
		if err != nil {
			return infos, err
		}

		for _, a := range node.Arguments {
			// TODO : we ignore the type since arguments have to particular type
			_, err := c.Compile(a)
			if err != nil {
				return infos, err
			}
		}

		c.emit(code.OpCall, len(node.Arguments))
	}

	return infos, err
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

func (c *Compiler) compileDeclare(
	nodeName string, nullable bool, nodeValue ast.Node, objectType object.ObjectType,
) error {
	infos, err := c.Compile(nodeValue)
	if err != nil {
		return err
	}

	if infos.Nullable && !nullable {
		return errorNullable(nodeName)
	}

	symbol := c.symbolTable.Define(
		nodeName,
		object.Attribute{ObjectType: objectType, Nullable: nullable},
	)

	if infos.ObjectType != symbol.ObjectInfo.ObjectType {
		return errorType(nodeName, symbol.ObjectInfo.ObjectType, infos.ObjectType)
	}
	if symbol.Scope == GlobalScope {
		c.emit(code.OpSetGlobal, symbol.Index)
	} else {
		c.emit(code.OpSetLocal, symbol.Index)
	}
	return nil
}

func errorUndefined(name string) error {
	return fmt.Errorf("undefined variable : '%s'", name)
}

func errorNullable(name string) error {
	return fmt.Errorf("null value error : '%s' is not nullable", name)
}

func errorType(name string, expected, got object.ObjectType) error {
	return fmt.Errorf("wrong type used : '%s' expect type '%s' but got '%s'", name, expected, got)
}

func errorCondition(objectType object.ObjectType) error {
	return fmt.Errorf("trying to use '%v' as condition", objectType)
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
