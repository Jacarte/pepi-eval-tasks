package jsonpatch_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"twelvemods/jsonpatch"
)

func m7Decode(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("bad test JSON %q: %v", s, err)
	}
	return v
}

func m7CheckPatchError(t *testing.T, what string, err error, index int, op, path string, sentinel error) {
	t.Helper()
	var pe *jsonpatch.PatchError
	if !errors.As(err, &pe) {
		t.Errorf("%s: error = %v, want *PatchError", what, err)
		return
	}
	// The spec only requires Err to satisfy errors.Is(Err, sentinel); a wrapped sentinel is fine.
	if pe.Index != index || pe.Op != op || pe.Path != path || !errors.Is(pe.Err, sentinel) {
		t.Errorf("%s: PatchError{%d, %q, %q, %v}, want {%d, %q, %q, Err satisfying errors.Is(Err, %v)}", what, pe.Index, pe.Op, pe.Path, pe.Err, index, op, path, sentinel)
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("%s: errors.Is(err, %v) is false", what, sentinel)
	}
}

func TestM7ParsePointer(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{"/", []string{""}},
		{"/a//b", []string{"a", "", "b"}},
		{"/a~1b", []string{"a/b"}},
		{"/m~0n", []string{"m~n"}},
		{"/~01", []string{"~1"}},
		{"/~10", []string{"/0"}},
		{"/foo/0/ bar", []string{"foo", "0", " bar"}},
	}
	for _, c := range cases {
		got, err := jsonpatch.ParsePointer(c.in)
		if err != nil {
			t.Errorf("ParsePointer(%q) error %v", c.in, err)
			continue
		}
		if len(got) != len(c.want) || (len(got) > 0 && !reflect.DeepEqual(got, c.want)) {
			t.Errorf("ParsePointer(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	for _, bad := range []string{"a", "#/a", "/~", "/~2", "/a~", "/a/~x"} {
		if _, err := jsonpatch.ParsePointer(bad); !errors.Is(err, jsonpatch.ErrInvalidPointer) {
			t.Errorf("ParsePointer(%q) error = %v, want ErrInvalidPointer", bad, err)
		}
	}
	formats := []struct {
		tokens []string
		want   string
	}{
		{nil, ""},
		{[]string{""}, "/"},
		{[]string{"a/b", "m~n"}, "/a~1b/m~0n"},
		{[]string{"~1"}, "/~01"},
		{[]string{"a", "0", ""}, "/a/0/"},
	}
	for _, f := range formats {
		if got := jsonpatch.FormatPointer(f.tokens); got != f.want {
			t.Errorf("FormatPointer(%q) = %q, want %q", f.tokens, got, f.want)
		}
	}
}

const m7RFCDoc = `{"foo": ["bar", "baz"], "": 0, "a/b": 1, "c%d": 2, "e^f": 3, "g|h": 4, "i\\j": 5, "k\"l": 6, " ": 7, "m~n": 8, "n": null, "o": {"p": [true, {"q": 1.5}]}}`

func TestM7Get(t *testing.T) {
	doc := m7Decode(t, m7RFCDoc)
	ok := []struct {
		ptr  string
		want string
	}{
		{"", m7RFCDoc},
		{"/foo", `["bar", "baz"]`},
		{"/foo/0", `"bar"`},
		{"/foo/1", `"baz"`},
		{"/", `0`},
		{"/a~1b", `1`},
		{"/c%d", `2`},
		{"/e^f", `3`},
		{"/g|h", `4`},
		{"/i\\j", `5`},
		{"/k\"l", `6`},
		{"/ ", `7`},
		{"/m~0n", `8`},
		{"/n", `null`},
		{"/o/p/1/q", `1.5`},
	}
	for _, c := range ok {
		got, err := jsonpatch.Get(doc, c.ptr)
		if err != nil {
			t.Errorf("Get(%q) error %v", c.ptr, err)
			continue
		}
		if want := m7Decode(t, c.want); !reflect.DeepEqual(got, want) {
			t.Errorf("Get(%q) = %#v, want %#v", c.ptr, got, want)
		}
	}
	bad := []struct {
		ptr  string
		want error
	}{
		{"/foo/2", jsonpatch.ErrPathNotFound},
		{"/foo/-", jsonpatch.ErrPathNotFound},
		{"/foo/01", jsonpatch.ErrInvalidIndex},
		{"/foo/x", jsonpatch.ErrInvalidIndex},
		{"/foo/-1", jsonpatch.ErrInvalidIndex},
		{"/foo/+1", jsonpatch.ErrInvalidIndex},
		{"/foo/", jsonpatch.ErrInvalidIndex},
		{"/foo/0/x", jsonpatch.ErrPathNotFound},
		{"/n/x", jsonpatch.ErrPathNotFound},
		{"/nope", jsonpatch.ErrPathNotFound},
		{"/foo/99999999999999999999999", jsonpatch.ErrPathNotFound},
		{"/FOO", jsonpatch.ErrPathNotFound},
		{"foo", jsonpatch.ErrInvalidPointer},
		{"/m~2n", jsonpatch.ErrInvalidPointer},
	}
	for _, c := range bad {
		if _, err := jsonpatch.Get(doc, c.ptr); !errors.Is(err, c.want) {
			t.Errorf("Get(%q) error = %v, want %v", c.ptr, err, c.want)
		}
	}
}

func TestM7Equal(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{`1`, `1.0`, true},
		{`1`, `"1"`, false},
		{`null`, `null`, true},
		{`null`, `false`, false},
		{`0`, `false`, false},
		{`""`, `null`, false},
		{`[1,2]`, `[1,2]`, true},
		{`[1,2]`, `[2,1]`, false},
		{`[1]`, `[1,1]`, false},
		{`{"a":1,"b":[true]}`, `{"b":[true],"a":1}`, true},
		{`{"a":1}`, `{"a":1,"b":null}`, false},
		{`{"a":null}`, `{"b":null}`, false},
		{`{}`, `[]`, false},
		{`"x"`, `"x"`, true},
	}
	for _, c := range cases {
		if got := jsonpatch.Equal(m7Decode(t, c.a), m7Decode(t, c.b)); got != c.want {
			t.Errorf("Equal(%s, %s) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

type m7Case struct {
	name, doc, patch string
	want             string // expected document when err is nil
	err              error
	index            int
	op, path         string
}

func m7Run(t *testing.T, cases []m7Case) {
	t.Helper()
	for _, c := range cases {
		got, err := jsonpatch.ApplyJSON([]byte(c.doc), []byte(c.patch))
		if c.err != nil {
			if got != nil {
				t.Errorf("%s: got document %s with an error", c.name, got)
			}
			m7CheckPatchError(t, c.name, err, c.index, c.op, c.path, c.err)
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
			continue
		}
		want, _ := json.Marshal(m7Decode(t, c.want))
		if string(got) != string(want) {
			t.Errorf("%s: got %s, want %s", c.name, got, want)
		}
	}
}

func TestM7RFCExamples(t *testing.T) {
	m7Run(t, []m7Case{
		{name: "A.1 add object member", doc: `{"foo":"bar"}`, patch: `[{"op":"add","path":"/baz","value":"qux"}]`, want: `{"baz":"qux","foo":"bar"}`},
		{name: "A.2 add array element", doc: `{"foo":["bar","baz"]}`, patch: `[{"op":"add","path":"/foo/1","value":"qux"}]`, want: `{"foo":["bar","qux","baz"]}`},
		{name: "A.3 remove member", doc: `{"baz":"qux","foo":"bar"}`, patch: `[{"op":"remove","path":"/baz"}]`, want: `{"foo":"bar"}`},
		{name: "A.4 remove element", doc: `{"foo":["bar","qux","baz"]}`, patch: `[{"op":"remove","path":"/foo/1"}]`, want: `{"foo":["bar","baz"]}`},
		{name: "A.5 replace", doc: `{"baz":"qux","foo":"bar"}`, patch: `[{"op":"replace","path":"/baz","value":"boo"}]`, want: `{"baz":"boo","foo":"bar"}`},
		{name: "A.6 move member", doc: `{"foo":{"bar":"baz","waldo":"fred"},"qux":{"corge":"grault"}}`, patch: `[{"op":"move","from":"/foo/waldo","path":"/qux/thud"}]`, want: `{"foo":{"bar":"baz"},"qux":{"corge":"grault","thud":"fred"}}`},
		{name: "A.7 move element", doc: `{"foo":["all","grass","cows","eat"]}`, patch: `[{"op":"move","from":"/foo/1","path":"/foo/3"}]`, want: `{"foo":["all","cows","eat","grass"]}`},
		{name: "A.8 test success", doc: `{"baz":"qux","foo":["a",2,"c"]}`, patch: `[{"op":"test","path":"/baz","value":"qux"},{"op":"test","path":"/foo/1","value":2}]`, want: `{"baz":"qux","foo":["a",2,"c"]}`},
		{name: "A.9 test failure", doc: `{"baz":"qux"}`, patch: `[{"op":"test","path":"/baz","value":"bar"}]`, err: jsonpatch.ErrTestFailed, index: 0, op: "test", path: "/baz"},
		{name: "A.10 add nested", doc: `{"foo":"bar"}`, patch: `[{"op":"add","path":"/child","value":{"grandchild":{}}}]`, want: `{"foo":"bar","child":{"grandchild":{}}}`},
		{name: "A.11 extra members", doc: `{"foo":"bar"}`, patch: `[{"op":"add","path":"/baz","value":"qux","xyz":123}]`, want: `{"foo":"bar","baz":"qux"}`},
		{name: "A.12 missing parent", doc: `{"foo":"bar"}`, patch: `[{"op":"add","path":"/baz/bat","value":"qux"}]`, err: jsonpatch.ErrPathNotFound, index: 0, op: "add", path: "/baz/bat"},
		{name: "A.14 escape order", doc: `{"/":9,"~1":10}`, patch: `[{"op":"test","path":"/~01","value":10}]`, want: `{"/":9,"~1":10}`},
		{name: "A.15 string vs number", doc: `{"/":9,"~1":10}`, patch: `[{"op":"test","path":"/~01","value":"10"}]`, err: jsonpatch.ErrTestFailed, index: 0, op: "test", path: "/~01"},
		{name: "A.16 append array value", doc: `{"foo":["bar"]}`, patch: `[{"op":"add","path":"/foo/-","value":["abc","def"]}]`, want: `{"foo":["bar",["abc","def"]]}`},
	})
}

func TestM7Operations(t *testing.T) {
	m7Run(t, []m7Case{
		{name: "add root", doc: `{"a":1}`, patch: `[{"op":"add","path":"","value":[1,2]}]`, want: `[1,2]`},
		{name: "add front", doc: `{"a":[1,2]}`, patch: `[{"op":"add","path":"/a/0","value":0}]`, want: `{"a":[0,1,2]}`},
		{name: "add at len", doc: `{"a":[1,2]}`, patch: `[{"op":"add","path":"/a/2","value":3}]`, want: `{"a":[1,2,3]}`},
		{name: "add past len", doc: `{"a":[1,2]}`, patch: `[{"op":"add","path":"/a/3","value":3}]`, err: jsonpatch.ErrPathNotFound, op: "add", path: "/a/3"},
		{name: "add leading zero", doc: `{"a":[1,2]}`, patch: `[{"op":"add","path":"/a/01","value":3}]`, err: jsonpatch.ErrInvalidIndex, op: "add", path: "/a/01"},
		{name: "add negative", doc: `{"a":[1,2]}`, patch: `[{"op":"add","path":"/a/-1","value":3}]`, err: jsonpatch.ErrInvalidIndex, op: "add", path: "/a/-1"},
		{name: "add word index", doc: `{"a":[1,2]}`, patch: `[{"op":"add","path":"/a/x","value":3}]`, err: jsonpatch.ErrInvalidIndex, op: "add", path: "/a/x"},
		{name: "add to scalar", doc: `{"a":1}`, patch: `[{"op":"add","path":"/a/b","value":2}]`, err: jsonpatch.ErrPathNotFound, op: "add", path: "/a/b"},
		{name: "add to null", doc: `{"a":null}`, patch: `[{"op":"add","path":"/a/b","value":2}]`, err: jsonpatch.ErrPathNotFound, op: "add", path: "/a/b"},
		{name: "add empty key", doc: `{}`, patch: `[{"op":"add","path":"/","value":1}]`, want: `{"":1}`},
		{name: "add replaces member", doc: `{"a":1}`, patch: `[{"op":"add","path":"/a","value":{"b":null}}]`, want: `{"a":{"b":null}}`},
		{name: "add escaped key", doc: `{}`, patch: `[{"op":"add","path":"/a~1b~0c","value":true}]`, want: `{"a/b~c":true}`},
		{name: "add dash key in object", doc: `{}`, patch: `[{"op":"add","path":"/-","value":1}]`, want: `{"-":1}`},
		{name: "add deep append", doc: `{"a":{"b":[[1]]}}`, patch: `[{"op":"add","path":"/a/b/0/-","value":2}]`, want: `{"a":{"b":[[1,2]]}}`},
		{name: "add null value", doc: `[]`, patch: `[{"op":"add","path":"/0","value":null}]`, want: `[null]`},
		{name: "remove last", doc: `[1,2,3]`, patch: `[{"op":"remove","path":"/2"}]`, want: `[1,2]`},
		{name: "remove dash", doc: `[1,2,3]`, patch: `[{"op":"remove","path":"/-"}]`, err: jsonpatch.ErrPathNotFound, op: "remove", path: "/-"},
		{name: "remove out of range", doc: `[1,2,3]`, patch: `[{"op":"remove","path":"/3"}]`, err: jsonpatch.ErrPathNotFound, op: "remove", path: "/3"},
		{name: "remove missing key", doc: `{"a":1}`, patch: `[{"op":"remove","path":"/b"}]`, err: jsonpatch.ErrPathNotFound, op: "remove", path: "/b"},
		{name: "remove root", doc: `{"a":1}`, patch: `[{"op":"remove","path":""}]`, err: jsonpatch.ErrInvalidOperation, op: "remove", path: ""},
		{name: "remove null member", doc: `{"a":null,"b":1}`, patch: `[{"op":"remove","path":"/a"}]`, want: `{"b":1}`},
		{name: "replace missing", doc: `{"a":1}`, patch: `[{"op":"replace","path":"/b","value":2}]`, err: jsonpatch.ErrPathNotFound, op: "replace", path: "/b"},
		{name: "replace root", doc: `{"a":1}`, patch: `[{"op":"replace","path":"","value":"x"}]`, want: `"x"`},
		{name: "replace dash", doc: `[1]`, patch: `[{"op":"replace","path":"/-","value":2}]`, err: jsonpatch.ErrPathNotFound, op: "replace", path: "/-"},
		{name: "replace element", doc: `[1,[2,3]]`, patch: `[{"op":"replace","path":"/1/0","value":null}]`, want: `[1,[null,3]]`},
		{name: "move into child", doc: `{"a":{"b":1}}`, patch: `[{"op":"move","from":"/a","path":"/a/c"}]`, err: jsonpatch.ErrInvalidOperation, op: "move", path: "/a/c"},
		{name: "move to sibling prefix", doc: `{"a":1}`, patch: `[{"op":"move","from":"/a","path":"/ab"}]`, want: `{"ab":1}`},
		{name: "move same path", doc: `{"a":[1,2]}`, patch: `[{"op":"move","from":"/a/0","path":"/a/0"}]`, want: `{"a":[1,2]}`},
		{name: "move same missing", doc: `{"a":[1,2]}`, patch: `[{"op":"move","from":"/a/5","path":"/a/5"}]`, err: jsonpatch.ErrPathNotFound, op: "move", path: "/a/5"},
		{name: "move missing", doc: `{}`, patch: `[{"op":"move","from":"/x","path":"/y"}]`, err: jsonpatch.ErrPathNotFound, op: "move", path: "/y"},
		{name: "move root", doc: `{"x":1}`, patch: `[{"op":"move","from":"","path":"/x"}]`, err: jsonpatch.ErrInvalidOperation, op: "move", path: "/x"},
		{name: "move to root", doc: `{"x":{"y":1}}`, patch: `[{"op":"move","from":"/x","path":""}]`, want: `{"y":1}`},
		{name: "move to end", doc: `{"a":[1,2,3]}`, patch: `[{"op":"move","from":"/a/0","path":"/a/-"}]`, want: `{"a":[2,3,1]}`},
		{name: "move past end after removal", doc: `{"a":[1,2,3]}`, patch: `[{"op":"move","from":"/a/0","path":"/a/3"}]`, err: jsonpatch.ErrPathNotFound, op: "move", path: "/a/3"},
		{name: "move between arrays", doc: `{"a":[1,2],"b":[]}`, patch: `[{"op":"move","from":"/a/1","path":"/b/0"}]`, want: `{"a":[1],"b":[2]}`},
		{name: "copy then edit copy", doc: `{"a":{"b":1}}`, patch: `[{"op":"copy","from":"/a","path":"/c"},{"op":"replace","path":"/c/b","value":2}]`, want: `{"a":{"b":1},"c":{"b":2}}`},
		{name: "copy root into itself", doc: `{"a":1}`, patch: `[{"op":"copy","from":"","path":"/self"}]`, want: `{"a":1,"self":{"a":1}}`},
		{name: "copy missing", doc: `{"a":1}`, patch: `[{"op":"copy","from":"/b","path":"/c"}]`, err: jsonpatch.ErrPathNotFound, op: "copy", path: "/c"},
		{name: "copy array element", doc: `[[1],2]`, patch: `[{"op":"copy","from":"/0","path":"/-"},{"op":"add","path":"/2/0","value":0}]`, want: `[[1],2,[0,1]]`},
		{name: "test root", doc: `{"a":[1,{"b":2}]}`, patch: `[{"op":"test","path":"","value":{"a":[1.0,{"b":2}]}}]`, want: `{"a":[1,{"b":2}]}`},
		{name: "test order matters", doc: `[1,2]`, patch: `[{"op":"test","path":"","value":[2,1]}]`, err: jsonpatch.ErrTestFailed, op: "test", path: ""},
		{name: "test missing", doc: `{}`, patch: `[{"op":"test","path":"/a","value":null}]`, err: jsonpatch.ErrPathNotFound, op: "test", path: "/a"},
		{name: "test null member", doc: `{"a":null}`, patch: `[{"op":"test","path":"/a","value":null}]`, want: `{"a":null}`},
		{name: "second op fails", doc: `{}`, patch: `[{"op":"add","path":"/x","value":1},{"op":"test","path":"/x","value":2}]`, err: jsonpatch.ErrTestFailed, index: 1, op: "test", path: "/x"},
		{name: "invalid path pointer", doc: `{}`, patch: `[{"op":"add","path":"x","value":1}]`, err: jsonpatch.ErrInvalidPointer, op: "add", path: "x"},
		{name: "invalid from pointer", doc: `{"a":1}`, patch: `[{"op":"copy","from":"a","path":"/b"}]`, err: jsonpatch.ErrInvalidPointer, op: "copy", path: "/b"},
		{name: "invalid path and from", doc: `{"a":1}`, patch: `[{"op":"move","from":"/~","path":"/~"}]`, err: jsonpatch.ErrInvalidPointer, op: "move", path: "/~"},
		{name: "empty patch", doc: `{"a":[1]}`, patch: `[]`, want: `{"a":[1]}`},
		{name: "sequence", doc: `{"list":[]}`, patch: `[{"op":"add","path":"/list/-","value":"a"},{"op":"add","path":"/list/0","value":"b"},{"op":"move","from":"/list/1","path":"/first"},{"op":"remove","path":"/list/0"},{"op":"test","path":"/list","value":[]}]`, want: `{"list":[],"first":"a"}`},
	})
}

func TestM7DecodePatch(t *testing.T) {
	ops, err := jsonpatch.DecodePatch([]byte(`[{"op":"add","path":"/a","value":{"b":[1,"x",null,true]}},{"op":"move","from":"/a","path":"/c"},{"op":"remove","path":"/c"},{"op":"test","path":"","value":null}]`))
	if err != nil {
		t.Fatalf("DecodePatch error %v", err)
	}
	want := []jsonpatch.Operation{
		{Op: "add", Path: "/a", Value: map[string]any{"b": []any{1.0, "x", nil, true}}},
		{Op: "move", Path: "/c", From: "/a"},
		{Op: "remove", Path: "/c"},
		{Op: "test", Path: "", Value: nil},
	}
	if !reflect.DeepEqual(ops, want) {
		t.Errorf("DecodePatch = %#v, want %#v", ops, want)
	}
	bad := []struct {
		patch    string
		index    int
		op, path string
	}{
		{`{"op":"add"}`, -1, "", ""},
		{`[{"op":"add","path":"/a","value":1}`, -1, "", ""},
		{`null`, -1, "", ""},
		{`[1]`, 0, "", ""},
		{`[{"op":1,"path":"/a"}]`, 0, "", "/a"},
		{`[{"path":"/a"}]`, 0, "", "/a"},
		{`[{"op":"merge","path":"/a"}]`, 0, "merge", "/a"},
		{`[{"op":"remove"}]`, 0, "remove", ""},
		{`[{"op":"remove","path":7}]`, 0, "remove", ""},
		{`[{"op":"add","path":"/a"}]`, 0, "add", "/a"},
		{`[{"op":"test","path":"/a"}]`, 0, "test", "/a"},
		{`[{"op":"replace","path":"/a"}]`, 0, "replace", "/a"},
		{`[{"op":"remove","path":"/a"},{"op":"copy","path":"/b"}]`, 1, "copy", "/b"},
		{`[{"op":"move","path":"/b","from":null}]`, 0, "move", "/b"},
		{`[{"op":"ADD","path":"/a","value":1}]`, 0, "ADD", "/a"},
	}
	for _, c := range bad {
		_, err := jsonpatch.DecodePatch([]byte(c.patch))
		m7CheckPatchError(t, "DecodePatch("+c.patch+")", err, c.index, c.op, c.path, jsonpatch.ErrInvalidOperation)
	}
	if ops, err := jsonpatch.DecodePatch([]byte(`[{"op":"add","path":"not a pointer","value":null}]`)); err != nil || len(ops) != 1 {
		t.Errorf("DecodePatch must not check pointers: ops=%v err=%v", ops, err)
	}
}

func TestM7ApplyJSONErrors(t *testing.T) {
	_, err := jsonpatch.ApplyJSON([]byte(`{"a":`), []byte(`[]`))
	if !errors.Is(err, jsonpatch.ErrInvalidDocument) {
		t.Errorf("invalid document: error = %v, want ErrInvalidDocument", err)
	}
	var pe *jsonpatch.PatchError
	if errors.As(err, &pe) {
		t.Errorf("invalid document: error must not be a *PatchError")
	}
	_, err = jsonpatch.ApplyJSON([]byte(`{}`), []byte(`[{"op":"bogus","path":"/a"}]`))
	m7CheckPatchError(t, "ApplyJSON bad patch", err, 0, "bogus", "/a", jsonpatch.ErrInvalidOperation)
}

func TestM7ApplyStructOperations(t *testing.T) {
	_, err := jsonpatch.Apply(map[string]any{}, []jsonpatch.Operation{{Op: "remove", Path: "/a"}, {Op: "nope", Path: "/a"}})
	m7CheckPatchError(t, "remove missing", err, 0, "remove", "/a", jsonpatch.ErrPathNotFound)
	_, err = jsonpatch.Apply(map[string]any{"a": 1.0}, []jsonpatch.Operation{{Op: "test", Path: "/a", Value: 1.0}, {Op: "nope", Path: "x"}})
	m7CheckPatchError(t, "unknown op checked first", err, 1, "nope", "x", jsonpatch.ErrInvalidOperation)
	got, err := jsonpatch.Apply(nil, []jsonpatch.Operation{{Op: "add", Path: "", Value: map[string]any{"k": "v"}}})
	if err != nil || !reflect.DeepEqual(got, map[string]any{"k": "v"}) {
		t.Errorf("add to nil document = %v, %v", got, err)
	}
}

func TestM7NonDestructive(t *testing.T) {
	src := `{"a":[1,2,3],"b":{"c":{"d":"e"}},"f":[[1],[2]]}`
	doc := m7Decode(t, src)
	value := map[string]any{"k": []any{1.0}}
	patch := []jsonpatch.Operation{
		{Op: "add", Path: "/a/1", Value: 9.0},
		{Op: "remove", Path: "/b/c/d"},
		{Op: "move", From: "/f/0", Path: "/f/-"},
		{Op: "add", Path: "/v", Value: value},
		{Op: "copy", From: "/f", Path: "/g"},
		{Op: "replace", Path: "/a/0", Value: "x"},
	}
	got, err := jsonpatch.Apply(doc, patch)
	if err != nil {
		t.Fatalf("Apply error %v", err)
	}
	want := m7Decode(t, `{"a":["x",9,2,3],"b":{"c":{}},"f":[[2],[1]],"v":{"k":[1]},"g":[[2],[1]]}`)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Apply = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(doc, m7Decode(t, src)) {
		t.Errorf("Apply modified its input document: %v", doc)
	}
	// The result must not share maps or slices with the input or the operation values.
	res := got.(map[string]any)
	res["v"].(map[string]any)["k"].([]any)[0] = "changed"
	if value["k"].([]any)[0] != 1.0 {
		t.Errorf("result shares a slice with an operation value")
	}
	res["g"].([]any)[0].([]any)[0] = "changed"
	if res["f"].([]any)[0].([]any)[0] != 2.0 {
		t.Errorf("copy shares a slice with its source")
	}
	got2, err := jsonpatch.Apply(doc, nil)
	if err != nil {
		t.Fatal(err)
	}
	got2.(map[string]any)["a"].([]any)[0] = "changed"
	got2.(map[string]any)["b"].(map[string]any)["new"] = true
	if !reflect.DeepEqual(doc, m7Decode(t, src)) {
		t.Errorf("result of an empty patch shares data with the input")
	}

	// A failing patch must leave the input untouched as well.
	failing := []jsonpatch.Operation{
		{Op: "remove", Path: "/a/0"},
		{Op: "add", Path: "/b/c/x", Value: 1.0},
		{Op: "replace", Path: "/f/0/0", Value: 5.0},
		{Op: "test", Path: "/a/0", Value: 1.0},
	}
	if out, err := jsonpatch.Apply(doc, failing); err == nil || out != nil {
		t.Fatalf("failing patch: got %v, %v", out, err)
	}
	if !reflect.DeepEqual(doc, m7Decode(t, src)) {
		t.Errorf("failing patch modified the input: %v", doc)
	}
}
