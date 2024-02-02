package compiler

import (
	"gold/object"
)

type SymbolScope string

const (
	LocalScope    SymbolScope = "LOCAL"
	GlobalScope   SymbolScope = "GLOBAL"
	BuiltinScope  SymbolScope = "BUILTIN"
	FreeScope     SymbolScope = "FREE"
	FunctionScope SymbolScope = "FUNCTION"
)

type Symbol struct {
	Name  string
	Scope SymbolScope

	// NOTE : number of symbol in the table. Index when the VM will
	// add the symbol in its list of symbol in the correct scope.
	Index      int
	ObjectInfo object.Attribute
}

type SymbolTable struct {
	Outer *SymbolTable

	store          map[string]Symbol
	numDefinitions int

	FreeSymbols []Symbol
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	return s
}

func NewSymbolTable() *SymbolTable {
	s := make(map[string]Symbol)
	free := []Symbol{}
	return &SymbolTable{store: s, FreeSymbols: free}
}

func (s *SymbolTable) Define(name string, objectInfo object.Attribute) Symbol {
	symbol := Symbol{Name: name, Index: s.numDefinitions, ObjectInfo: objectInfo}
	if s.Outer == nil {
		symbol.Scope = GlobalScope
	} else {
		symbol.Scope = LocalScope
	}

	s.store[name] = symbol
	s.numDefinitions++
	return symbol
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	if !ok && s.Outer != nil {
		obj, ok = s.Outer.Resolve(name)
		if !ok {
			return obj, ok
		}

		if obj.Scope == GlobalScope || obj.Scope == BuiltinScope {
			return obj, ok
		}

		free := s.defineFree(obj, obj.ObjectInfo)
		return free, true
	}
	return obj, ok
}

func (s *SymbolTable) DefineBuiltin(index int, name string, objectInfo object.Attribute) Symbol {
	symbol := Symbol{Name: name, Index: index, Scope: BuiltinScope, ObjectInfo: objectInfo}
	s.store[name] = symbol
	return symbol
}

func (s *SymbolTable) DefineFunctionName(name string, objectInfo object.Attribute) Symbol {
	symbol := Symbol{Name: name, Index: 0, Scope: FunctionScope, ObjectInfo: objectInfo}
	s.store[name] = symbol
	return symbol
}

func (s *SymbolTable) defineFree(original Symbol, objectInfo object.Attribute) Symbol {
	s.FreeSymbols = append(s.FreeSymbols, original)

	symbol := Symbol{Name: original.Name, Index: len(s.FreeSymbols) - 1, ObjectInfo: objectInfo}
	symbol.Scope = FreeScope

	s.store[original.Name] = symbol
	return symbol
}
