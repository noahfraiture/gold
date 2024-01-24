package compiler

import "testing"

func TestDefine(t *testing.T) {
	expected := map[string]Symbol{
		"a": {Name: "a", Scope: GlobalScope, Index: 0, Nullable: true},
		"b": {Name: "b", Scope: GlobalScope, Index: 1, Nullable: true},
		"c": {Name: "c", Scope: LocalScope, Index: 0, Nullable: true},
		"d": {Name: "d", Scope: LocalScope, Index: 1, Nullable: true},
		"e": {Name: "e", Scope: LocalScope, Index: 0, Nullable: true},
		"f": {Name: "f", Scope: LocalScope, Index: 1, Nullable: true},
	}

	global := NewSymbolTable()

	a := global.Define("a", true)
	if a != expected["a"] {
		t.Errorf("expected a=%+v, got=%+v", expected["a"], a)
	}

	b := global.Define("b", true)
	if b != expected["b"] {
		t.Errorf("expected b=%+v, got=%+v", expected["b"], b)
	}

	firstLocal := NewEnclosedSymbolTable(global)

	c := firstLocal.Define("c", true)
	if c != expected["c"] {
		t.Errorf("expected c=%+v, got=%+v", expected["c"], c)
	}

	d := firstLocal.Define("d", true)
	if d != expected["d"] {
		t.Errorf("expected d=%+v, got=%+v", expected["d"], d)
	}

	secondLocal := NewEnclosedSymbolTable(firstLocal)

	e := secondLocal.Define("e", true)
	if e != expected["e"] {
		t.Errorf("expected e=%+v, got=%+v", expected["e"], e)
	}

	f := secondLocal.Define("f", true)
	if f != expected["f"] {
		t.Errorf("expected f=%+v, got=%+v", expected["f"], f)
	}
}

func TestResolveGlobal(t *testing.T) {
	global := NewSymbolTable()
	global.Define("a", true)
	global.Define("b", true)

	expected := []Symbol{
		{Name: "a", Scope: GlobalScope, Index: 0, Nullable: true},
		{Name: "b", Scope: GlobalScope, Index: 1, Nullable: true},
	}

	for _, sym := range expected {
		result, ok := global.Resolve(sym.Name)
		if !ok {
			t.Errorf("name %s not resolvable", sym.Name)
			continue
		}
		if result != sym {
			t.Errorf("expected %s to resolve to %+v, got=%+v",
				sym.Name, sym, result)
		}
	}
}

func TestResolveLocal(t *testing.T) {
	global := NewSymbolTable()
	global.Define("a", true)
	global.Define("b", true)

	local := NewEnclosedSymbolTable(global)
	local.Define("c", true)
	local.Define("d", true)

	expected := []Symbol{
		{Name: "a", Scope: GlobalScope, Index: 0, Nullable: true},
		{Name: "b", Scope: GlobalScope, Index: 1, Nullable: true},
		{Name: "c", Scope: LocalScope, Index: 0, Nullable: true},
		{Name: "d", Scope: LocalScope, Index: 1, Nullable: true},
	}

	for _, sym := range expected {
		result, ok := local.Resolve(sym.Name)
		if !ok {
			t.Errorf("name %s not resolvable", sym.Name)
			continue
		}
		if result != sym {
			t.Errorf("expected %s to resolve to %+v, got=%+v",
				sym.Name, sym, result)
		}
	}
}

func TestResolveNestedLocal(t *testing.T) {
	global := NewSymbolTable()
	global.Define("a", true)
	global.Define("b", true)

	firstLocal := NewEnclosedSymbolTable(global)
	firstLocal.Define("c", true)
	firstLocal.Define("d", true)

	secondLocal := NewEnclosedSymbolTable(firstLocal)
	secondLocal.Define("e", true)
	secondLocal.Define("f", true)

	tests := []struct {
		table           *SymbolTable
		expectedSymbols []Symbol
	}{
		{
			firstLocal,
			[]Symbol{
				{Name: "a", Scope: GlobalScope, Index: 0, Nullable: true},
				{Name: "b", Scope: GlobalScope, Index: 1, Nullable: true},
				{Name: "c", Scope: LocalScope, Index: 0, Nullable: true},
				{Name: "d", Scope: LocalScope, Index: 1, Nullable: true},
			},
		},
		{
			secondLocal,
			[]Symbol{
				{Name: "a", Scope: GlobalScope, Index: 0, Nullable: true},
				{Name: "b", Scope: GlobalScope, Index: 1, Nullable: true},
				{Name: "e", Scope: LocalScope, Index: 0, Nullable: true},
				{Name: "f", Scope: LocalScope, Index: 1, Nullable: true},
			},
		},
	}

	for _, tt := range tests {
		for _, sym := range tt.expectedSymbols {
			result, ok := tt.table.Resolve(sym.Name)
			if !ok {
				t.Errorf("name %s not resolvable", sym.Name)
				continue
			}
			if result != sym {
				t.Errorf("expected %s to resolve to %+v, got=%+v",
					sym.Name, sym, result)
			}
		}
	}
}

func TestDefineResolveBuiltins(t *testing.T) {
	global := NewSymbolTable()
	firstLocal := NewEnclosedSymbolTable(global)
	secondLocal := NewEnclosedSymbolTable(firstLocal)

	expected := []Symbol{
		{Name: "a", Scope: BuiltinScope, Index: 0},
		{Name: "c", Scope: BuiltinScope, Index: 1},
		{Name: "e", Scope: BuiltinScope, Index: 2},
		{Name: "f", Scope: BuiltinScope, Index: 3},
	}

	for i, v := range expected {
		global.DefineBuiltin(i, v.Name)
	}

	for _, table := range []*SymbolTable{global, firstLocal, secondLocal} {
		for _, sym := range expected {
			result, ok := table.Resolve(sym.Name)
			if !ok {
				t.Errorf("name %s not resolvable", sym.Name)
				continue
			}
			if result != sym {
				t.Errorf("expected %s to resolve to %+v, got=%+v",
					sym.Name, sym, result)
			}
		}
	}
}

func TestResolveFree(t *testing.T) {
	global := NewSymbolTable()
	global.Define("a", true)
	global.Define("b", true)

	firstLocal := NewEnclosedSymbolTable(global)
	firstLocal.Define("c", true) // TODO : everything to true
	firstLocal.Define("d", true)

	secondLocal := NewEnclosedSymbolTable(firstLocal)
	secondLocal.Define("e", true)
	secondLocal.Define("f", true)

	tests := []struct {
		table               *SymbolTable
		expectedSymbols     []Symbol
		expectedFreeSymbols []Symbol
	}{
		{
			firstLocal,
			[]Symbol{
				{Name: "a", Scope: GlobalScope, Index: 0, Nullable: true},
				{Name: "b", Scope: GlobalScope, Index: 1, Nullable: true},
				{Name: "c", Scope: LocalScope, Index: 0, Nullable: true},
				{Name: "d", Scope: LocalScope, Index: 1, Nullable: true},
			},
			[]Symbol{},
		},
		{
			secondLocal,
			[]Symbol{
				{Name: "a", Scope: GlobalScope, Index: 0, Nullable: true},
				{Name: "b", Scope: GlobalScope, Index: 1, Nullable: true},
				{Name: "c", Scope: FreeScope, Index: 0, Nullable: true},
				{Name: "d", Scope: FreeScope, Index: 1, Nullable: true},
				{Name: "e", Scope: LocalScope, Index: 0, Nullable: true},
				{Name: "f", Scope: LocalScope, Index: 1, Nullable: true},
			},
			[]Symbol{
				{Name: "c", Scope: LocalScope, Index: 0, Nullable: true},
				{Name: "d", Scope: LocalScope, Index: 1, Nullable: true},
			},
		},
	}

	for _, tt := range tests {
		for _, sym := range tt.expectedSymbols {
			result, ok := tt.table.Resolve(sym.Name)
			if !ok {
				t.Errorf("name %s not resolvable", sym.Name)
				continue
			}
			if result != sym {
				t.Errorf("expected %s to resolve to %+v, got=%+v",
					sym.Name, sym, result)
			}
		}

		if len(tt.table.FreeSymbols) != len(tt.expectedFreeSymbols) {
			t.Errorf("wrong number of free symbols. got=%d, want=%d",
				len(tt.table.FreeSymbols), len(tt.expectedFreeSymbols))
			continue
		}

		for i, sym := range tt.expectedFreeSymbols {
			result := tt.table.FreeSymbols[i]
			if result != sym {
				t.Errorf("wrong free symbol. got=%+v, want=%+v",
					result, sym)
			}
		}
	}
}

func TestResolveUnresolvableFree(t *testing.T) {
	global := NewSymbolTable()
	global.Define("a", true)

	firstLocal := NewEnclosedSymbolTable(global)
	firstLocal.Define("c", true)

	secondLocal := NewEnclosedSymbolTable(firstLocal)
	secondLocal.Define("e", true)
	secondLocal.Define("f", true)

	expected := []Symbol{
		{Name: "a", Scope: GlobalScope, Index: 0, Nullable: true},
		{Name: "c", Scope: FreeScope, Index: 0, Nullable: true},
		{Name: "e", Scope: LocalScope, Index: 0, Nullable: true},
		{Name: "f", Scope: LocalScope, Index: 1, Nullable: true},
	}

	for _, sym := range expected {
		result, ok := secondLocal.Resolve(sym.Name)
		if !ok {
			t.Errorf("name %s not resolvable", sym.Name)
			continue
		}
		if result != sym {
			t.Errorf("expected %s to resolve to %+v, got=%+v",
				sym.Name, sym, result)
		}
	}

	expectedUnresolvable := []string{
		"b",
		"d",
	}

	for _, name := range expectedUnresolvable {
		_, ok := secondLocal.Resolve(name)
		if ok {
			t.Errorf("name %s resolved, but was expected not to", name)
		}
	}
}

func TestDefineAndResolveFunctionName(t *testing.T) {
	global := NewSymbolTable()
	global.DefineFunctionName("a")

	expected := Symbol{Name: "a", Scope: FunctionScope, Index: 0}

	result, ok := global.Resolve(expected.Name)
	if !ok {
		t.Fatalf("function name %s not resolvable", expected.Name)
	}

	if result != expected {
		t.Errorf("expected %s to resolve to %+v, got=%+v",
			expected.Name, expected, result)
	}
}

func TestShadowingFunctionName(t *testing.T) {
	global := NewSymbolTable()
	global.DefineFunctionName("a")
	global.Define("a", true)

	expected := Symbol{Name: "a", Scope: GlobalScope, Index: 0, Nullable: true}

	result, ok := global.Resolve(expected.Name)
	if !ok {
		t.Fatalf("function name %s not resolvable", expected.Name)
	}

	if result != expected {
		t.Errorf("expected %s to resolve to %+v, got=%+v",
			expected.Name, expected, result)
	}
}
