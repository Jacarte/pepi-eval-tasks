package expr_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"twelvemods/expr"
)

func TestM10String(t *testing.T) {
	cases := []struct{ src, want string }{
		{"1+2*3", "(1 + (2 * 3))"},
		{"1-2-3", "((1 - 2) - 3)"},
		{"8/4/2", "((8 / 4) / 2)"},
		{"2^3^2", "(2 ^ (3 ^ 2))"},
		{"-2^2", "(-(2 ^ 2))"},
		{"2^-1", "(2 ^ (-1))"},
		{"-a ^ -b ^ c", "(-(a ^ (-(b ^ c))))"},
		{"--x", "(-(-x))"},
		{"!a && b || c", "(((!a) && b) || c)"},
		{"a || b && c", "(a || (b && c))"},
		{"a == b != c", "((a == b) != c)"},
		{"a < b == c > d", "((a < b) == (c > d))"},
		{"!x == y", "((!x) == y)"},
		{"a + -b * c", "(a + ((-b) * c))"},
		{"a ? b : c ? d : e", "(a ? b : (c ? d : e))"},
		{"a ? b ? c : d : e", "(a ? (b ? c : d) : e)"},
		{"a || b ? c + 1 : d", "((a || b) ? (c + 1) : d)"},
		{"(1 + 2) * 3", "((1 + 2) * 3)"},
		{"((x))", "x"},
		{"x % 3 / 2 * 1", "(((x % 3) / 2) * 1)"},
		{"max(1, 2+3, f(x))", "max(1, (2 + 3), f(x))"},
		{"len()", "len()"},
		{`"a\"b\n" + "\\"`, `("a\"b\n" + "\\")`},
		{"2.0", "2.0"},
		{"1e3", "1000.0"},
		{"1.5e300", "1.5e+300"},
		{"0.1", "0.1"},
		{"1E-7", "1e-07"},
		{"12.50", "12.5"},
		{"007", "7"},
		{"true && false", "(true && false)"},
		{"9223372036854775807", "9223372036854775807"},
		{"1\n+\t2", "(1 + 2)"},
		{"1+2*3 == 7 && !ok ? max(a, 2) : -x^2", "((((1 + (2 * 3)) == 7) && (!ok)) ? max(a, 2) : (-(x ^ 2)))"},
		{"a<=b>=c", "((a <= b) >= c)"},
	}
	for _, c := range cases {
		e, err := expr.Parse(c.src)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.src, err)
			continue
		}
		if got := e.String(); got != c.want {
			t.Errorf("Parse(%q).String() = %q, want %q", c.src, got, c.want)
		}
	}
}

func TestM10Eval(t *testing.T) {
	env := map[string]any{
		"a": int64(1), "b": 2.5, "c": int64(2), "x": int64(5), "f": 1.5,
		"name": "hi", "yes": true, "no": false, "n": int64(21),
	}
	cases := []struct {
		src  string
		want any
	}{
		{"1 + 2 * 3", int64(7)},
		{"7 / 2", int64(3)},
		{"-7 / 2", int64(-3)},
		{"7 % 3", int64(1)},
		{"-7 % 3", int64(-1)},
		{"7 % -3", int64(1)},
		{"7.0 / 2", 3.5},
		{"1 + 2.5", 3.5},
		{"2 * 1.5", 3.0},
		{"10 - 0.5", 9.5},
		{"6 / 3.0", 2.0},
		{"2 ^ 10", int64(1024)},
		{"2 ^ 0", int64(1)},
		{"0 ^ 0", int64(1)},
		{"2 ^ -1", 0.5},
		{"2.0 ^ 3", 8.0},
		{"4 ^ 0.5", 2.0},
		{"-2 ^ 2", int64(-4)},
		{"(-2) ^ 3", int64(-8)},
		{"2 ^ 3 ^ 2", int64(512)},
		{"2 ^ 63", int64(math.MinInt64)},
		{"2 ^ 64", int64(0)},
		{"1 ^ 9223372036854775807", int64(1)},
		{"9223372036854775807 + 1", int64(math.MinInt64)},
		{"-9223372036854775807 - 1", int64(math.MinInt64)},
		{"-(-9223372036854775807 - 1)", int64(math.MinInt64)},
		{"4611686018427387904 * 2", int64(math.MinInt64)},
		{`"ab" + "cd"`, "abcd"},
		{`"a" < "b"`, true},
		{`"B" < "a"`, true},
		{`"abc" >= "abd"`, false},
		{`"" < "a"`, true},
		{"1 < 2.5", true},
		{"2 <= 2", true},
		{"3 > 3.0", false},
		{"3 >= 3.0", true},
		{"1 == 1.0", true},
		{"1 != 1.0", false},
		{"9007199254740993 == 9007199254740992", false},
		{"9007199254740993 > 9007199254740992", true},
		{`"1" == 1`, false},
		{"true == 1", false},
		{`true != "true"`, true},
		{"true == true", true},
		{`"a" == "a"`, true},
		{"true && false", false},
		{"true || false", true},
		{"!true", false},
		{"!!true", true},
		{"false && 1/0 == 1", false},
		{"true || undefinedVar", true},
		{"no || yes", true},
		{"yes && no", false},
		{`x > 1 ? "big" : "small"`, "big"},
		{"true ? 1 : 1/0", int64(1)},
		{"false ? 1/0 : 2", int64(2)},
		{"false ? 1 : true ? 2 : 3", int64(2)},
		{"a + b * c", 6.0},
		{`name + "!"`, "hi!"},
		{"f ^ 2", 2.25},
		{"n * 2", int64(42)},
		{"min(3, 1, 2)", int64(1)},
		{"max(3, 1, 2)", int64(3)},
		{"min(1, 2.5)", 1.0},
		{"max(3, 2.5)", 3.0},
		{"min(5)", int64(5)},
		{"max(-1, -2)", int64(-1)},
		{"abs(-3)", int64(3)},
		{"abs(-2.5)", 2.5},
		{"abs(3)", int64(3)},
		{"abs(-9223372036854775807 - 1)", int64(math.MinInt64)},
		{`len("héllo")`, int64(5)},
		{`len("")`, int64(0)},
		{`upper("abc") + lower("DeF")`, "ABCdef"},
		{`len(name + "xyz") * 2`, int64(10)},
		{`"a\tb"`, "a\tb"},
		{"\"line\nbreak\"", "line\nbreak"},
		{"1e3", 1000.0},
		{"1E+2", 100.0},
		{"2.5e-1", 0.25},
		{"(1 + 2) * 3", int64(9)},
		{"1 + 2 == 3 && 4 > 3", true},
		{"-x", int64(-5)},
		{"-f", -1.5},
	}
	for _, c := range cases {
		got, err := expr.Eval(c.src, env)
		if err != nil {
			t.Errorf("Eval(%q): %v", c.src, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Eval(%q) = %#v (%T), want %#v (%T)", c.src, got, got, c.want, c.want)
		}
	}
}

func TestM10ReuseAndNilEnv(t *testing.T) {
	e, err := expr.Parse("x * 2 + 1")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		x    any
		want any
	}{{int64(1), int64(3)}, {0.5, 2.0}, {int64(-4), int64(-7)}} {
		got, err := e.Eval(map[string]any{"x": c.x})
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("Eval(x=%v) = %#v, %v; want %#v", c.x, got, err, c.want)
		}
	}
	if got, err := expr.Eval("1 + 1", nil); err != nil || got != int64(2) {
		t.Errorf("Eval with nil env = %#v, %v", got, err)
	}
}

type m10Err struct {
	src       string
	line, col int
	msg       string
}

func m10CheckErr(t *testing.T, what string, err error, c m10Err) {
	t.Helper()
	var ee *expr.Error
	if !errors.As(err, &ee) {
		t.Errorf("%s(%q) error = %v, want *expr.Error", what, c.src, err)
		return
	}
	if ee.Line != c.line || ee.Column != c.col || ee.Msg != c.msg {
		t.Errorf("%s(%q) = %d:%d %q, want %d:%d %q", what, c.src, ee.Line, ee.Column, ee.Msg, c.line, c.col, c.msg)
	}
}

func TestM10ParseErrors(t *testing.T) {
	cases := []m10Err{
		{"", 1, 1, "unexpected end of input"},
		{"1 +", 1, 4, "unexpected end of input"},
		{"1 +\n", 2, 1, "unexpected end of input"},
		{"1 + ", 1, 5, "unexpected end of input"},
		{"(1 + 2", 1, 7, "unexpected end of input"},
		{"-", 1, 2, "unexpected end of input"},
		{"a ? b", 1, 6, "unexpected end of input"},
		{"f(1,", 1, 5, "unexpected end of input"},
		{"1 2", 1, 3, `unexpected token "2"`},
		{"1 + * 2", 1, 5, `unexpected token "*"`},
		{")", 1, 1, `unexpected token ")"`},
		{"()", 1, 2, `unexpected token ")"`},
		{"f(1,)", 1, 5, `unexpected token ")"`},
		{"f(1 2)", 1, 5, `unexpected token "2"`},
		{"a ? b c", 1, 7, `unexpected token "c"`},
		{"a : b", 1, 3, `unexpected token ":"`},
		{`"a" "b"`, 1, 5, `unexpected token "\"b\""`},
		{"1e", 1, 2, `unexpected token "e"`},
		{"1e+", 1, 2, `unexpected token "e"`},
		{"true()", 1, 5, `unexpected token "("`},
		{"1 <= >= 2", 1, 6, `unexpected token ">="`},
		{"\t)", 1, 2, `unexpected token ")"`},
		{"x\n  + \n  )", 3, 3, `unexpected token ")"`},
		{"1 & 2", 1, 3, "unexpected character '&'"},
		{"a = b", 1, 3, "unexpected character '='"},
		{"1 | 2", 1, 3, "unexpected character '|'"},
		{"x.y", 1, 2, "unexpected character '.'"},
		{"1.", 1, 2, "unexpected character '.'"},
		{".5", 1, 1, "unexpected character '.'"},
		{"é", 1, 1, "unexpected character 'é'"},
		{"1 +\n  @", 2, 3, "unexpected character '@'"},
		{"2 3 &", 1, 5, "unexpected character '&'"},
		{`"abc`, 1, 1, "unterminated string"},
		{`1 + "ab\`, 1, 5, "unterminated string"},
		{`"a\qb"`, 1, 3, `invalid escape \q`},
		{"x +\n\"ab\ncd\\x\"", 3, 3, `invalid escape \x`},
		{`"é\r"`, 1, 3, `invalid escape \r`},
		{"9223372036854775808", 1, 1, "integer literal out of range"},
		{"1 + 99999999999999999999", 1, 5, "integer literal out of range"},
		{"1e999", 1, 1, "float literal out of range"},
	}
	for _, c := range cases {
		e, err := expr.Parse(c.src)
		if e != nil {
			t.Errorf("Parse(%q) returned an expression with an error", c.src)
		}
		m10CheckErr(t, "Parse", err, c)
	}
	_, err := expr.Parse("1 +")
	if err == nil || err.Error() != "1:4: unexpected end of input" {
		t.Errorf("Error() = %v", err)
	}
}

func TestM10EvalErrors(t *testing.T) {
	env := map[string]any{"s": "str", "i": int64(3)}
	cases := []m10Err{
		{`1 + "a"`, 1, 3, `invalid operands for "+": int and string`},
		{`"a" - "b"`, 1, 5, `invalid operands for "-": string and string`},
		{"true * 2", 1, 6, `invalid operands for "*": bool and int`},
		{"1.5 % 2", 1, 5, `invalid operands for "%": float and int`},
		{`"a" < 1`, 1, 5, `invalid operands for "<": string and int`},
		{"true < false", 1, 6, `invalid operands for "<": bool and bool`},
		{`2 ^ "a"`, 1, 3, `invalid operands for "^": int and string`},
		{"s / 2", 1, 3, `invalid operands for "/": string and int`},
		{"1 / 0", 1, 3, "division by zero"},
		{"1.0 / 0.0", 1, 5, "division by zero"},
		{"1 / 0.0", 1, 3, "division by zero"},
		{"5 % 0", 1, 3, "division by zero"},
		{`-"a"`, 1, 1, `invalid operand for "-": string`},
		{"!1", 1, 1, `invalid operand for "!": int`},
		{"1 && true", 1, 3, `invalid operand for "&&": int`},
		{"true && 1", 1, 6, `invalid operand for "&&": int`},
		{`false || "x"`, 1, 7, `invalid operand for "||": string`},
		{"i ? 2 : 3", 1, 3, `invalid operand for "?": int`},
		{"x + 1", 1, 1, `undefined variable "x"`},
		{"1 +\n  y", 2, 3, `undefined variable "y"`},
		{"foo(1)", 1, 1, `unknown function "foo"`},
		{" foo(1/0)", 1, 2, `unknown function "foo"`},
		{"min()", 1, 1, `wrong number of arguments for "min"`},
		{"abs(1, 2)", 1, 1, `wrong number of arguments for "abs"`},
		{"1 + len()", 1, 5, `wrong number of arguments for "len"`},
		{"upper(1/0, 2)", 1, 8, "division by zero"},
		{"len(5)", 1, 1, `invalid argument for "len": int`},
		{`abs("x")`, 1, 1, `invalid argument for "abs": string`},
		{`max(1, "a", true)`, 1, 1, `invalid argument for "max": string`},
		{"lower(true)", 1, 1, `invalid argument for "lower": bool`},
		{"(1 + true) && false", 1, 4, `invalid operands for "+": int and bool`},
		{`1 + 2 + (3 - "z")`, 1, 12, `invalid operands for "-": int and string`},
		{"(1/0) + (2 % 0)", 1, 3, "division by zero"},
		{"i > 2 ? s * 2 : 0", 1, 11, `invalid operands for "*": string and int`},
	}
	for _, c := range cases {
		e, err := expr.Parse(c.src)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.src, err)
			continue
		}
		v, err := e.Eval(env)
		if v != nil {
			t.Errorf("Eval(%q) returned %#v with an error", c.src, v)
		}
		m10CheckErr(t, "Eval", err, c)
	}
}
