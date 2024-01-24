package vm

import (
	"fmt"
	"gold/ast"
	"gold/compiler"
	"gold/lexer"
	"gold/object"
	"gold/parser"
	"testing"
)

func TestIntegerArithmetic(t *testing.T) {
	tests := []vmTestCase{
		{"1", 1},
		{"2", 2},
		{"1 + 2", 3},
		{"1 - 2", -1},
		{"1 * 2", 2},
		{"4 / 2", 2},
		{"50 / 2 * 2 + 10 - 5", 55},
		{"5 * (2 + 10)", 60},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"5 * 2 + 10", 20},
		{"5 + 2 * 10", 25},
		{"5 * (2 + 10)", 60},
		{"-5", -5},
		{"-10", -10},
		{"-50 + 100 + -50", 0},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
	}

	runVmTests(t, tests)
}

func TestFloatArithmetic(t *testing.T) {
	tests := []vmTestCase{
		{"1.0", 1.0},
		{"2.5", 2.5},
		{"1.0 + 2", 3.0},
		{"1 - 2.5", -1.5},
		{"1.5 * 2.0", 3.0},
		{"4.0 / 2.0", 2.0},
		{"50.0 / 2 * 2.5 + 10 - 5", 67.5},
		{"5.5 * (2 + 10)", 66.0},
		{"5.0 + 5 + 5 + 5 - 10", 10.0},
		{"2 * 2 * 2 * 2.0 * 2", 32.0},
		{"5.0 * 2 + 10", 20.0},
		{"5 + 2.5 * 10", 30.0},
		{"5.0 * (2.5 + 10)", 62.5},
		{"-5.0", -5.0},
		{"-10.", -10.0},
		{"-50.0 + 100 + -50.5", -0.5},
		{"(5.0 + 10 * 2.5 + 15 / 3.0) * 2 + -10", 60.0},
	}

	runVmTests(t, tests)
}

func TestBooleanExpressions(t *testing.T) {
	tests := []vmTestCase{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 < 1", false},
		{"1 > 1", false},
		{"1 <= 2", true},
		{"1 >= 2", false},
		{"1 <= 1", true},
		{"1 >= 1", true},
		{"1 == 1", true},
		{"1 != 1", false},
		{"1 == 2", false},
		{"1 != 2", true},
		{"1.1 < 2.2", true},
		{"1.1 > 2.2", false},
		{"1.1 < 1.1", false},
		{"1.1 > 1.1", false},
		{"1.1 <= 2.2", true},
		{"1.1 >= 2.2", false},
		{"1.1 <= 1.1", true},
		{"1.1 >= 1.1", true},
		{"1.1 == 1.1", true},
		{"1.1 != 1.1", false},
		{"1.1 == 2.2", false},
		{"1.1 != 2.2", true},
		{"true == true", true},
		{"false == false", true},
		{"true == false", false},
		{"true != false", true},
		{"false != true", true},
		{"(1 < 2) == true", true},
		{"(1 < 2) == false", false},
		{"(1 > 2) == true", false},
		{"(1 > 2) == false", true},
		{"!true", false},
		{"!false", true},
		{"!5", false},
		{"!2.3", false},
		{"!!0", false},
		{"!!0.0", false},
		{"!!true", true},
		{"!!false", false},
		{"!!5", true},
		{"!!5.", true},
		{"!(if (false) { 5; })", true},
	}

	runVmTests(t, tests)
}

func TestConditionals(t *testing.T) {
	tests := []vmTestCase{
		{"if (true) { 10 }", 10},
		{"if (true) { 10 } else { 20 }", 10},
		{"if (false) { 10 } else { 20 } ", 20},
		{"if (1) { 10 }", 10},
		{"if (1 < 2) { 10 }", 10},
		{"if (1 <= 2) { 11 } else { 20 }", 11},
		{"if (1 >= 2) { 12 } else { 20 }", 20},
		{"if (1 > 2) { 10 }", Null},
		{"if (false) { 10 }", Null},
		{"if ((if (false) { 10 })) { 10 } else { 20 }", 20},
		{"if (true) {}", Null}, // The last popped value is the conditional
	}

	runVmTests(t, tests)
}

func TestWhile(t *testing.T) {
	tests := []vmTestCase{
		{"while (false) {10}", Null},
		{"let x = 0; while (x < 10) {x++}", Null}, // While loop produce a Null value since there are expressions
		{"let x = 0; while (x < 10) {x = x + 1}", Null},
		{"let x = 0; while (x < 10) {x = x + 1} x", 10},
		{"let x = 0; while (x < 10) {if (x == 8) {x = x + 20} else {x++}} x", 28},
	}

	runVmTests(t, tests)
}

func TestIncDecExpressions(t *testing.T) {
	tests := []vmTestCase{
		{"let x = 0; x++", 0},
		{"let x = 0; x--", 0},
		{"let x = 0; x++; x", 1},
		{"let x = 0; x--; x", -1},
		{"let x = 0; ++x", 1},
		{"let x = 0; --x", -1},
		{"let x = 0; ++x; x", 1},
		{"let x = 0; --x; x", -1},
	}

	runVmTests(t, tests)
}

func TestGlobalLetStatements(t *testing.T) {
	tests := []vmTestCase{
		{"let one = 1; one", 1},
		{"let one = 1; let two = 2; one + two", 3},
		{"let one = 1; let two = one + one; one + two", 3},
	}

	runVmTests(t, tests)
}

func TestGlobalReassignStatements(t *testing.T) {
	tests := []vmTestCase{
		{"let one = 1; one = 2; one", 2},
		{"let one = 1; let two = 2; one = one + two; one", 3},
		{"let one = 1; one = one + one; one + one", 4},
	}

	runVmTests(t, tests)
}

func TestStringExpressions(t *testing.T) {
	tests := []vmTestCase{
		{`"gold"`, "gold"},
		{`"go" + "ld"`, "gold"},
		{`"go" + "ld" + "money"`, "goldmoney"},
	}

	runVmTests(t, tests)
}

// TODO : ensure array can have everything
func TestArrayLiterals(t *testing.T) {
	tests := []vmTestCase{
		{"[]", []int{}},
		{"[1, 2, 3]", []int{1, 2, 3}},
		{"[1 + 2, 3 * 4, 5 + 6]", []int{3, 12, 11}},
	}

	runVmTests(t, tests)
}

func TestHashLiterals(t *testing.T) {
	tests := []vmTestCase{
		{
			"{}", map[object.HashKey]int64{},
		},
		{
			"{1: 2, 2: 3}",
			map[object.HashKey]int64{
				(&object.Integer{Value: 1}).HashKey(): 2,
				(&object.Integer{Value: 2}).HashKey(): 3,
			},
		},
		{
			"{1 + 1: 2 * 2, 3 + 3: 4 * 4}",
			map[object.HashKey]int64{
				(&object.Integer{Value: 2}).HashKey(): 4,
				(&object.Integer{Value: 6}).HashKey(): 16,
			},
		},
	}

	runVmTests(t, tests)
}

func TestIndexExpressions(t *testing.T) {
	tests := []vmTestCase{
		{"[1, 2, 3][1]", 2},
		{"[1, 2, 3][0 + 2]", 3},
		{"[[1, 1, 1]][0][0]", 1},
		{"[][0]", Null},
		{"[1, 2, 3][99]", Null},
		{"[1][-1]", Null},
		{"{1: 1, 2: 2}[1]", 1},
		{"{1: 1, 2: 2}[2]", 2},
		{"{1: 1}[0]", Null},
		{"{}[0]", Null},
	}

	runVmTests(t, tests)
}

func TestCallingFunctionsWithoutArguments(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
		let fivePlusTen = fn() { 5 + 10; };
		fivePlusTen();
		`,
			expected: 15,
		},
		{
			input: `
		let one = fn() { 1; };
		let two = fn() { 2; };
		one() + two()
		`,
			expected: 3,
		},
		{
			input: `
		let a = fn() { 1 };
		let b = fn() { a() + 1 };
		let c = fn() { b() + 1 };
		c();
		`,
			expected: 3,
		},
	}

	runVmTests(t, tests)
}

func TestFunctionsWithReturnStatement(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
		let earlyExit = fn() { return 99; 100; };
		earlyExit();
		`,
			expected: 99,
		},
		{
			input: `
		let earlyExit = fn() { return 99; return 100; };
		earlyExit();
		`,
			expected: 99,
		},
	}

	runVmTests(t, tests)
}

func TestFunctionsWithoutReturnValue(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
		let noReturn = fn() { };
		noReturn();
		`,
			expected: Null,
		},
		{
			input: `
		let noReturn = fn() { };
		let noReturnTwo = fn() { noReturn(); };
		noReturn();
		noReturnTwo();
		`,
			expected: Null,
		},
	}

	runVmTests(t, tests)
}

func TestFirstClassFunctions(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
		let returnsOne = fn() { 1; };
		let returnsOneReturner = fn() { returnsOne; };
		returnsOneReturner()();
		`,
			expected: 1,
		},
		{
			input: `
		let returnsOneReturner = fn() {
			let returnsOne = fn() { 1; };
			returnsOne;
		};
		returnsOneReturner()();
		`,
			expected: 1,
		},
	}

	runVmTests(t, tests)
}

func TestCallingFunctionsWithBindings(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
		let one = fn() { let one = 1; one };
		one();
		`,
			expected: 1,
		},
		{
			input: `
		let oneAndTwo = fn() { let one = 1; let two = 2; one + two; };
		oneAndTwo();
		`,
			expected: 3,
		},
		{
			input: `
		let oneAndTwo = fn() { let one = 1; let two = 2; one + two; };
		let threeAndFour = fn() { let three = 3; let four = 4; three + four; };
		oneAndTwo() + threeAndFour();
		`,
			expected: 10,
		},
		{
			input: `
		let firstFoobar = fn() { let foobar = 50; foobar; };
		let secondFoobar = fn() { let foobar = 100; foobar; };
		firstFoobar() + secondFoobar();
		`,
			expected: 150,
		},
		{
			input: `
		let globalSeed = 50;
		let minusOne = fn() {
			let num = 1;
			globalSeed - num;
		}
		let minusTwo = fn() {
			let num = 2;
			globalSeed - num;
		}
		minusOne() + minusTwo();
		`,
			expected: 97,
		},
	}

	runVmTests(t, tests)
}

func TestCallingFunctionsWithArgumentsAndBindings(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
		may identity = fn(a) { a; };
		identity(4);
		`,
			expected: 4,
		},
		{
			input: `
		may sum = fn(a, b) { a + b; };
		sum(1, 2);
		`,
			expected: 3,
		},
		{
			input: `
		may sum = fn(a, b) {
			may c = a + b;
			c;
		};
		sum(1, 2);
		`,
			expected: 3,
		},
		{
			input: `
		may sum = fn(a, b) {
			may c = a + b;
			c;
		};
		sum(1, 2) + sum(3, 4);`,
			expected: 10,
		},
		{
			input: `
		may sum = fn(a, b) {
			may c = a + b;
			c;
		};
		may outer = fn() {
			sum(1, 2) + sum(3, 4);
		};
		outer();
		`,
			expected: 10,
		},
		{
			input: `
		let globalNum = 10;

		may sum = fn(a, b) {
			may c = a + b;
			c + globalNum;
		};

		may outer = fn() {
			sum(1, 2) + sum(3, 4) + globalNum;
		};

		outer() + globalNum;
		`,
			expected: 50,
		},
	}

	runVmTests(t, tests)
}

func TestCallingFunctionsWithWrongArguments(t *testing.T) {
	tests := []vmTestCase{
		{
			input:    `fn() { 1; }(1);`,
			expected: `wrong number of arguments: want=0, got=1`,
		},
		{
			input:    `fn(a) { a; }();`,
			expected: `wrong number of arguments: want=1, got=0`,
		},
		{
			input:    `fn(a, b) { a + b; }(1);`,
			expected: `wrong number of arguments: want=2, got=1`,
		},
	}

	for _, tt := range tests {
		program := parse(tt.input)

		comp := compiler.New()
		err, _ := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		vm := New(comp.Bytecode())
		err = vm.Run()
		if err == nil {
			t.Fatalf("expected VM error but resulted in none.")
		}

		if err.Error() != tt.expected {
			t.Fatalf("wrong VM error: want=%q, got=%q", tt.expected, err)
		}
	}
}

func TestBuiltinFunctions(t *testing.T) {
	tests := []vmTestCase{
		{`len("")`, 0},
		{`len("four")`, 4},
		{`len("hello world")`, 11},
		{
			`len(1)`,
			&object.Error{
				Message: "argument to `len` not supported, got INTEGER",
			},
		},
		{
			`len("one", "two")`,
			&object.Error{
				Message: "wrong number of arguments. got=2, want=1",
			},
		},
		{`len([1, 2, 3])`, 3},
		{`len([])`, 0},
		{`puts("hello", "world!")`, Null},
		{`first([1, 2, 3])`, 1},
		{`first([])`, Null},
		{
			`first(1)`,
			&object.Error{
				Message: "argument to `first` must be ARRAY, got INTEGER",
			},
		},
		{`last([1, 2, 3])`, 3},
		{`last([])`, Null},
		{
			`last(1)`,
			&object.Error{
				Message: "argument to `last` must be ARRAY, got INTEGER",
			},
		},
		{`rest([1, 2, 3])`, []int{2, 3}},
		{`rest([])`, Null},
		{`push([], 1)`, []int{1}},
		{
			`push(1, 1)`,
			&object.Error{
				Message: "argument to `push` must be ARRAY, got INTEGER",
			},
		},
	}

	runVmTests(t, tests)
}

func TestClosures(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
		may newClosure = fn(a) {
			fn() { a; };
		};
		may closure = newClosure(99);
		closure();
		`,
			expected: 99,
		},
		{
			input: `
		may newAdder = fn(a, b) {
			fn(c) { a + b + c };
		};
		may adder = newAdder(1, 2);
		adder(8);
		`,
			expected: 11,
		},
		{
			input: `
		may newAdder = fn(a, b) {
			may c = a + b;
			fn(d) { c + d };
		};
		may adder = newAdder(1, 2);
		adder(8);
		`,
			expected: 11,
		},
		{
			input: `
		may newAdderOuter = fn(a, b) {
			may c = a + b;
			fn(d) {
				may e = d + c;
				fn(f) { e + f; };
			};
		};
		may newAdderInner = newAdderOuter(1, 2)
		may adder = newAdderInner(3);
		adder(8);
		`,
			expected: 14,
		},
		{
			input: `
		let a = 1;
		may newAdderOuter = fn(b) {
			fn(c) {
				fn(d) { a + b + c + d };
			};
		};
		may newAdderInner = newAdderOuter(2)
		may adder = newAdderInner(3);
		adder(8);
		`,
			expected: 14,
		},
		{
			input: `
		may newClosure = fn(a, b) {
			may one = fn() { a; };
			may two = fn() { b; };
			fn() { one() + two(); };
		};
		may closure = newClosure(9, 90);
		closure();
		`,
			expected: 99,
		},
	}

	runVmTests(t, tests)
}

func TestRecursiveFunctions(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
		let countDown = fn(x) {
			if (x == 0) {
				return 0;
			} else {
				countDown(x - 1);
			}
		};
		countDown(1);
		`,
			expected: 0,
		},
		{
			input: `
		let countDown = fn(x) {
			if (x == 0) {
				return 0;
			} else {
				countDown(x - 1);
			}
		};
		let wrapper = fn() {
			countDown(1);
		};
		wrapper();
		`,
			expected: 0,
		},
		{
			input: `
		let wrapper = fn() {
			let countDown = fn(x) {
				if (x == 0) {
					return 0;
				} else {
					countDown(x - 1);
				}
			};
			countDown(1);
		};
		wrapper();
		`,
			expected: 0,
		},
	}

	runVmTests(t, tests)
}

func TestRecursiveFibonacci(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
		let fibonacci = fn(x) {
			if (x == 0) {
				return 0;
			} else {
				if (x == 1) {
					return 1;
				} else {
					fibonacci(x - 1) + fibonacci(x - 2);
				}
			}
		};
		fibonacci(15);
		`,
			expected: 610,
		},
	}

	runVmTests(t, tests)
}

type vmTestCase struct {
	input    string
	expected interface{}
}

func runVmTests(t *testing.T, tests []vmTestCase) {
	t.Helper()

	for _, tt := range tests {
		program := parse(tt.input)

		comp := compiler.New()
		err, _ := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		vm := New(comp.Bytecode())
		err = vm.Run()
		if err != nil {
			t.Fatalf("vm error: %s", err)
		}

		stackElem := vm.LastPoppedStackElem()

		testExpectedObject(t, tt.expected, stackElem)
	}
}

func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func testExpectedObject(
	t *testing.T,
	expected interface{},
	actual object.Object,
) {
	t.Helper()

	switch expected := expected.(type) {
	case int:
		err := testIntegerObject(int64(expected), actual)
		if err != nil {
			t.Errorf("testIntegerObject failed: %s", err)
		}

	case float64:
		err := testFloatObject(float64(expected), actual)
		if err != nil {
			t.Errorf("testFloatObject failed: %s", err)
		}

	case bool:
		err := testBooleanObject(bool(expected), actual)
		if err != nil {
			t.Errorf("testBooleanObject failed: %s", err)
		}

	case *object.Null:
		if actual != Null {
			t.Errorf("object is not Null: %T (%+v)", actual, actual)
		}

	case string:
		err := testStringObject(expected, actual)
		if err != nil {
			t.Errorf("testStringObject failed: %s", err)
		}

	case []int:
		array, ok := actual.(*object.Array)
		if !ok {
			t.Errorf("object not Array: %T (%+v)", actual, actual)
			return
		}

		if len(array.Elements) != len(expected) {
			t.Errorf("wrong num of elements. want=%d, got=%d",
				len(expected), len(array.Elements))
			return
		}

		for i, expectedElem := range expected {
			err := testIntegerObject(int64(expectedElem), array.Elements[i])
			if err != nil {
				t.Errorf("testIntegerObject failed: %s", err)
			}
		}

	case map[object.HashKey]int64:
		hash, ok := actual.(*object.Hash)
		if !ok {
			t.Errorf("object is not Hash. got=%T (%+v)", actual, actual)
			return
		}

		if len(hash.Pairs) != len(expected) {
			t.Errorf("hash has wrong number of Pairs. want=%d, got=%d",
				len(expected), len(hash.Pairs))
			return
		}

		for expectedKey, expectedValue := range expected {
			pair, ok := hash.Pairs[expectedKey]
			if !ok {
				t.Errorf("no pair for given key in Pairs")
			}

			err := testIntegerObject(expectedValue, pair.Value)
			if err != nil {
				t.Errorf("testIntegerObject failed: %s", err)
			}
		}

	case *object.Error:
		errObj, ok := actual.(*object.Error)
		if !ok {
			t.Errorf("object is not Error: %T (%+v)", actual, actual)
			return
		}
		if errObj.Message != expected.Message {
			t.Errorf("wrong error message. expected=%q, got=%q",
				expected.Message, errObj.Message)
		}
	}
}

func testIntegerObject(expected int64, actual object.Object) error {
	result, ok := actual.(*object.Integer)
	if !ok {
		return fmt.Errorf("object is not Integer. got=%T (%+v)",
			actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value. got=%d, want=%d",
			result.Value, expected)
	}

	return nil
}

func testFloatObject(expected float64, actual object.Object) error {
	result, ok := actual.(*object.Float)
	if !ok {
		return fmt.Errorf("object is not Float. got=%T (%+v)",
			actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value. got=%f, want=%f",
			result.Value, expected)
	}

	return nil
}

func testBooleanObject(expected bool, actual object.Object) error {
	result, ok := actual.(*object.Boolean)
	if !ok {
		return fmt.Errorf("object is not Boolean. got=%T (%+v)",
			actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value. got=%t, want=%t",
			result.Value, expected)
	}

	return nil
}

func testStringObject(expected string, actual object.Object) error {
	result, ok := actual.(*object.String)
	if !ok {
		return fmt.Errorf("object is not String. got=%T (%+v)",
			actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value. got=%q, want=%q",
			result.Value, expected)
	}

	return nil
}
