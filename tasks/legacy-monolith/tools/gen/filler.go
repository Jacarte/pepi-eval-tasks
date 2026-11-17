package main

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"
	"strings"
	"unicode"
)

// style holds the per-package choices that make generated packages look like
// they were written by different people at different times.
type style struct {
	h          helperNames
	typedValid bool   // ValidationError vs fmt.Errorf(ErrInvalid)
	rw         bool   // sync.RWMutex vs sync.Mutex
	strict     bool   // codec rejects unknown keys
	recv       string // receiver name for memory stores
	who        []string
}

type helperNames struct {
	esc, unesc, split, cut, kv, rawkv, ft, pt string
	replacer                                  bool
}

var helperSets = []helperNames{
	{"escapeField", "unescapeField", "splitEscaped", "cutEscaped", "writeKV", "writeRawKV", "formatTime", "parseTime", false},
	{"esc", "unesc", "splitUnescaped", "cutUnescaped", "putPair", "putRawPair", "fmtTime", "parseTimeField", true},
	{"quoteValue", "unquoteValue", "splitOn", "cutOn", "appendField", "appendRawField", "timeString", "timeValue", false},
}

var authors = []string{"mk", "ravi", "jlo", "dana", "pc", "tobias", "s.ng", "ana", "oskar", "lw"}

func rngFor(name string) *rand.Rand {
	h := fnv.New64a()
	h.Write([]byte(name))
	return rand.New(rand.NewSource(int64(h.Sum64()&0x7fffffffffff) + 1987))
}

func styleFor(p Pkg, r *rand.Rand) style {
	st := style{
		h:          helperSets[r.Intn(len(helperSets))],
		typedValid: r.Intn(2) == 0,
		rw:         r.Intn(3) != 0,
		strict:     r.Intn(2) == 0,
		recv:       []string{"m", "s", "st"}[r.Intn(3)],
	}
	perm := r.Perm(len(authors))
	for _, k := range perm[:3] {
		st.who = append(st.who, authors[k])
	}
	return st
}

type storeNames struct{ get, put, del, list string }

func storeFor(styleName string) storeNames {
	switch styleName {
	case "find":
		return storeNames{"Find", "Save", "Remove", "All"}
	case "load":
		return storeNames{"Load", "Store", "Drop", "Scan"}
	default:
		return storeNames{"Get", "Put", "Delete", "List"}
	}
}

// ---- small naming helpers ----

func lowerFirst(s string) string {
	// BinSlot -> binSlot, SKU -> sku, ID -> id
	rs := []rune(s)
	n := 0
	for n < len(rs) && unicode.IsUpper(rs[n]) {
		n++
	}
	if n == 0 {
		return s
	}
	if n == 1 || n == len(rs) {
		return strings.ToLower(string(rs[:n])) + string(rs[n:])
	}
	return strings.ToLower(string(rs[:n-1])) + string(rs[n-1:])
}

func snake(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(rs[i-1]) || (i+1 < len(rs) && unicode.IsLower(rs[i+1]))) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func words(s string) string { return strings.ReplaceAll(snake(s), "_", " ") }

func pascal(snakeName string) string {
	parts := strings.Split(snakeName, "_")
	for k, p := range parts {
		if p == "" {
			continue
		}
		parts[k] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func plural(s string) string {
	switch {
	case strings.HasSuffix(s, "y") && !strings.HasSuffix(s, "ey"):
		return s[:len(s)-1] + "ies"
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "ch"):
		return s + "es"
	}
	return s + "s"
}

func goType(f Field) string {
	switch f.Kind {
	case kString:
		return "string"
	case kInt:
		return "int"
	case kInt64:
		return "int64"
	case kCents:
		return "money.Cents"
	case kBool:
		return "bool"
	case kTime:
		return "time.Time"
	case kTags:
		return "[]string"
	}
	panic("unknown kind " + f.Kind)
}

func numeric(f Field) bool { return f.Kind == kInt || f.Kind == kInt64 || f.Kind == kCents }

func hasKind(e Entity, k string) bool {
	for _, f := range e.Fields {
		if f.Kind == k {
			return true
		}
	}
	return false
}

func firstField(e Entity, pred func(Field) bool) (Field, bool) {
	for _, f := range e.Fields {
		if pred(f) {
			return f, true
		}
	}
	return Field{}, false
}

// writer is a tiny line-oriented builder.
type writer struct{ b strings.Builder }

func (w *writer) p(format string, a ...any) {
	fmt.Fprintf(&w.b, format, a...)
	w.b.WriteByte('\n')
}
func (w *writer) raw(s string)   { w.b.WriteString(s) }
func (w *writer) String() string { return w.b.String() }

func (st style) todo(r *rand.Rand) string {
	notes := []string{
		"drop this once the v1 admin screens are gone",
		"this should probably live in a shared package",
		"copy of the helper in another package; keep them in sync",
		"add metrics here",
		"revisit after the 2021 data migration",
		"the export job depends on this exact behaviour",
		"not sure this is still needed",
		"make this configurable",
		"handle the legacy upper-case values from the old importer",
		"remove the special case below when ops confirms",
	}
	tickets := []string{"OPS-1423", "SHOP-311", "SHOP-902", "INFRA-77", "DATA-1180", "SHOP-1544", "FIN-208"}
	who := st.who[r.Intn(len(st.who))]
	n := notes[r.Intn(len(notes))]
	switch r.Intn(4) {
	case 0:
		return fmt.Sprintf("// TODO(%s): %s.", who, n)
	case 1:
		return fmt.Sprintf("// TODO(%s): %s (%s).", who, n, tickets[r.Intn(len(tickets))])
	case 2:
		return fmt.Sprintf("// FIXME: %s.", n)
	default:
		return fmt.Sprintf("// NOTE(%s): %s.", who, n)
	}
}

// ---- package level files ----

func genErrors(p Pkg, st style) string {
	w := &writer{}
	w.p("package %s", p.Name)
	w.p("")
	w.p("@@IMPORTS@@")
	w.p("var (")
	w.p("\t// ErrNotFound is returned when a %s record does not exist.", p.Name)
	w.p("\tErrNotFound = errors.New(%q)", p.Name+": not found")
	w.p("\t// ErrInvalid marks validation failures. Use errors.Is to test for it.")
	w.p("\tErrInvalid = errors.New(%q)", p.Name+": invalid")
	w.p(")")
	if st.typedValid {
		w.p("")
		w.p("// ValidationError describes a single invalid field. It unwraps to ErrInvalid.")
		w.p("type ValidationError struct {")
		w.p("\tEntity string")
		w.p("\tField  string")
		w.p("\tReason string")
		w.p("}")
		w.p("")
		w.p("func (e *ValidationError) Error() string {")
		w.p("\treturn fmt.Sprintf(\"%s: invalid %%s.%%s: %%s\", e.Entity, e.Field, e.Reason)", p.Name)
		w.p("}")
		w.p("")
		w.p("func (e *ValidationError) Unwrap() error { return ErrInvalid }")
	}
	return w.String()
}

func genUtil(p Pkg, st style, r *rand.Rand) string {
	h := st.h
	w := &writer{}
	w.p("package %s", p.Name)
	w.p("")
	w.p("@@IMPORTS@@")
	w.p("// FieldChange is one differing field between two versions of a record.")
	w.p("// The admin UI renders these in the history tab.")
	w.p("type FieldChange struct {")
	w.p("\tField string")
	w.p("\tFrom  string")
	w.p("\tTo    string")
	w.p("}")
	w.p("")
	if h.replacer {
		w.p("var fieldEscaper = strings.NewReplacer(\"\\\\\", \"\\\\\\\\\", \";\", \"\\\\;\", \"=\", \"\\\\=\", \",\", \"\\\\,\")")
		w.p("")
		w.p("// %s escapes the separators of the key=value wire format.", h.esc)
		w.p("func %s(v string) string { return fieldEscaper.Replace(v) }", h.esc)
	} else {
		w.p("// %s escapes the separators of the key=value wire format.", h.esc)
		w.p("func %s(v string) string {", h.esc)
		w.p("\tif !strings.ContainsAny(v, \"\\\\;=,\") {")
		w.p("\t\treturn v")
		w.p("\t}")
		w.p("\tvar b strings.Builder")
		w.p("\tfor i := 0; i < len(v); i++ {")
		w.p("\t\tswitch v[i] {")
		w.p("\t\tcase '\\\\', ';', '=', ',':")
		w.p("\t\t\tb.WriteByte('\\\\')")
		w.p("\t\t}")
		w.p("\t\tb.WriteByte(v[i])")
		w.p("\t}")
		w.p("\treturn b.String()")
		w.p("}")
	}
	w.p("")
	w.p("func %s(v string) string {", h.unesc)
	w.p("\tif !strings.Contains(v, \"\\\\\") {")
	w.p("\t\treturn v")
	w.p("\t}")
	w.p("\tvar b strings.Builder")
	w.p("\tfor i := 0; i < len(v); i++ {")
	w.p("\t\tif v[i] == '\\\\' && i+1 < len(v) {")
	w.p("\t\t\ti++")
	w.p("\t\t}")
	w.p("\t\tb.WriteByte(v[i])")
	w.p("\t}")
	w.p("\treturn b.String()")
	w.p("}")
	w.p("")
	w.p("// %s splits v on sep, ignoring escaped separators. Parts stay escaped.", h.split)
	w.p("func %s(v string, sep byte) []string {", h.split)
	w.p("\tvar out []string")
	w.p("\tstart := 0")
	w.p("\tfor i := 0; i < len(v); i++ {")
	w.p("\t\tif v[i] == '\\\\' {")
	w.p("\t\t\ti++")
	w.p("\t\t\tcontinue")
	w.p("\t\t}")
	w.p("\t\tif v[i] == sep {")
	w.p("\t\t\tout = append(out, v[start:i])")
	w.p("\t\t\tstart = i + 1")
	w.p("\t\t}")
	w.p("\t}")
	w.p("\treturn append(out, v[start:])")
	w.p("}")
	w.p("")
	w.p("func %s(v string, sep byte) (before, after string, found bool) {", h.cut)
	w.p("\tfor i := 0; i < len(v); i++ {")
	w.p("\t\tif v[i] == '\\\\' {")
	w.p("\t\t\ti++")
	w.p("\t\t\tcontinue")
	w.p("\t\t}")
	w.p("\t\tif v[i] == sep {")
	w.p("\t\t\treturn v[:i], v[i+1:], true")
	w.p("\t\t}")
	w.p("\t}")
	w.p("\treturn v, \"\", false")
	w.p("}")
	w.p("")
	w.p("func %s(b *strings.Builder, key, value string) { %s(b, key, %s(value)) }", h.kv, h.rawkv, h.esc)
	w.p("")
	w.p("func %s(b *strings.Builder, key, raw string) {", h.rawkv)
	w.p("\tif b.Len() > 0 {")
	w.p("\t\tb.WriteByte(';')")
	w.p("\t}")
	w.p("\tb.WriteString(key)")
	w.p("\tb.WriteByte('=')")
	w.p("\tb.WriteString(raw)")
	w.p("}")
	w.p("")
	w.p("func %s(t time.Time) string {", h.ft)
	w.p("\tif t.IsZero() {")
	w.p("\t\treturn \"\"")
	w.p("\t}")
	w.p("\treturn t.UTC().Format(time.RFC3339Nano)")
	w.p("}")
	w.p("")
	w.p("func %s(v string) (time.Time, error) {", h.pt)
	w.p("\tif v == \"\" {")
	w.p("\t\treturn time.Time{}, nil")
	w.p("\t}")
	w.p("\treturn time.Parse(time.RFC3339Nano, v)")
	w.p("}")
	// A couple of the duplicated helpers every legacy package grows.
	pool := []func(){
		func() {
			w.p("")
			w.p("// trimLower normalises user supplied codes.")
			w.p("%s", st.todo(r))
			w.p("func trimLower(v string) string { return strings.ToLower(strings.TrimSpace(v)) }")
		},
		func() {
			w.p("")
			w.p("func firstNonEmpty(vals ...string) string {")
			w.p("\tfor _, v := range vals {")
			w.p("\t\tif strings.TrimSpace(v) != \"\" {")
			w.p("\t\t\treturn v")
			w.p("\t\t}")
			w.p("\t}")
			w.p("\treturn \"\"")
			w.p("}")
		},
		func() {
			w.p("")
			w.p("func clampInt(v, lo, hi int) int {")
			w.p("\tif v < lo {")
			w.p("\t\treturn lo")
			w.p("\t}")
			w.p("\tif v > hi {")
			w.p("\t\treturn hi")
			w.p("\t}")
			w.p("\treturn v")
			w.p("}")
		},
		func() {
			w.p("")
			w.p("func containsFold(list []string, v string) bool {")
			w.p("\tfor _, s := range list {")
			w.p("\t\tif strings.EqualFold(s, v) {")
			w.p("\t\t\treturn true")
			w.p("\t\t}")
			w.p("\t}")
			w.p("\treturn false")
			w.p("}")
		},
	}
	for _, k := range r.Perm(len(pool))[:2+r.Intn(2)] {
		pool[k]()
	}
	return w.String()
}

// ---- per entity files ----

type ectx struct {
	p   Pkg
	e   Entity
	st  style
	sn  storeNames
	r   *rand.Rand
	low string // lowerFirst(e.Name)
	hum string // words(e.Name)
}

func (x *ectx) statusConst(s string) string { return x.e.Name + "Status" + pascal(s) }

func (x *ectx) invalid(field, reason string) string {
	if x.st.typedValid {
		return fmt.Sprintf("&ValidationError{Entity: %q, Field: %q, Reason: %q}", snake(x.e.Name), field, reason)
	}
	return fmt.Sprintf("fmt.Errorf(\"%%w: %s.%s %s\", ErrInvalid)", snake(x.e.Name), field, reason)
}

func genModel(x *ectx) string {
	e, w := x.e, &writer{}
	w.p("package %s", x.p.Name)
	w.p("")
	w.p("@@IMPORTS@@")
	w.p("// %s %s", e.Name, e.Doc)
	if x.r.Intn(3) == 0 {
		w.p("//")
		w.p("// The zero value is not valid; build one with all required fields and run")
		w.p("// Validate before handing it to a store.")
	}
	w.p("type %s struct {", e.Name)
	w.p("\tID string")
	for _, f := range e.Fields {
		if f.Doc != "" {
			w.p("\t%s %s // %s", f.Name, goType(f), f.Doc)
		} else {
			w.p("\t%s %s", f.Name, goType(f))
		}
	}
	w.p("\tStatus    %sStatus", e.Name)
	w.p("\tCreatedAt time.Time")
	w.p("\tUpdatedAt time.Time")
	w.p("}")
	w.p("")
	w.p("// %sStatus is the lifecycle state of a %s.", e.Name, x.hum)
	w.p("type %sStatus int", e.Name)
	w.p("")
	w.p("const (")
	w.p("\t%s %sStatus = iota", x.statusConst("unknown"), e.Name)
	for _, s := range e.Statuses {
		w.p("\t%s", x.statusConst(s))
	}
	w.p(")")
	w.p("")
	names := []string{"\"unknown\""}
	for _, s := range e.Statuses {
		names = append(names, fmt.Sprintf("%q", s))
	}
	w.p("var %sStatusNames = []string{%s}", x.low, strings.Join(names, ", "))
	w.p("")
	w.p("func (s %sStatus) String() string {", e.Name)
	w.p("\tif s < 0 || int(s) >= len(%sStatusNames) {", x.low)
	w.p("\t\treturn fmt.Sprintf(\"%sStatus(%%d)\", int(s))", e.Name)
	w.p("\t}")
	w.p("\treturn %sStatusNames[s]", x.low)
	w.p("}")
	w.p("")
	w.p("// Parse%sStatus parses the wire name of a status. Matching is", e.Name)
	w.p("// case-insensitive because the old importer wrote upper-case values.")
	w.p("func Parse%sStatus(v string) (%sStatus, error) {", e.Name, e.Name)
	w.p("\tv = strings.ToLower(strings.TrimSpace(v))")
	w.p("\tfor i, n := range %sStatusNames {", x.low)
	w.p("\t\tif i > 0 && n == v {")
	w.p("\t\t\treturn %sStatus(i), nil", e.Name)
	w.p("\t\t}")
	w.p("\t}")
	w.p("\treturn %s, fmt.Errorf(\"%%w: unknown %s status %%q\", ErrInvalid, v)", x.statusConst("unknown"), x.hum)
	w.p("}")
	w.p("")
	w.p("var %sTransitions = map[%sStatus][]%sStatus{", x.low, e.Name, e.Name)
	n := len(e.Statuses)
	for k := 0; k < n-1; k++ {
		to := []string{x.statusConst(e.Statuses[k+1])}
		if k+1 != n-1 {
			to = append(to, x.statusConst(e.Statuses[n-1]))
		}
		w.p("\t%s: {%s},", x.statusConst(e.Statuses[k]), strings.Join(to, ", "))
	}
	w.p("}")
	w.p("")
	w.p("// CanTransition reports whether a %s may move from s to next.", x.hum)
	w.p("func (s %sStatus) CanTransition(next %sStatus) bool {", e.Name, e.Name)
	w.p("\tfor _, t := range %sTransitions[s] {", x.low)
	w.p("\t\tif t == next {")
	w.p("\t\t\treturn true")
	w.p("\t\t}")
	w.p("\t}")
	w.p("\treturn false")
	w.p("}")
	w.p("")
	w.p("// Terminal reports whether no further transitions are possible.")
	w.p("func (s %sStatus) Terminal() bool { return len(%sTransitions[s]) == 0 }", e.Name, x.low)
	w.p("")
	w.p("// Validate checks the invariants every stored %s must satisfy.", x.hum)
	w.p("func (%s *%s) Validate() error {", "v", e.Name)
	w.p("\tif strings.TrimSpace(v.ID) == \"\" {")
	w.p("\t\treturn %s", x.invalid("id", "is required"))
	w.p("\t}")
	for _, f := range e.Fields {
		switch {
		case f.Kind == kString && f.Required:
			w.p("\tif strings.TrimSpace(v.%s) == \"\" {", f.Name)
			w.p("\t\treturn %s", x.invalid(snake(f.Name), "is required"))
			w.p("\t}")
		case numeric(f) && !f.Signed:
			w.p("\tif v.%s < 0 {", f.Name)
			w.p("\t\treturn %s", x.invalid(snake(f.Name), "must not be negative"))
			w.p("\t}")
		case f.Kind == kTags:
			w.p("\tfor _, tag := range v.%s {", f.Name)
			w.p("\t\tif strings.TrimSpace(tag) == \"\" {")
			w.p("\t\t\treturn %s", x.invalid(snake(f.Name), "contains an empty tag"))
			w.p("\t\t}")
			w.p("\t}")
		}
	}
	w.p("\tif v.Status <= %s || int(v.Status) >= len(%sStatusNames) {", x.statusConst("unknown"), x.low)
	w.p("\t\treturn %s", x.invalid("status", "is not a known status"))
	w.p("\t}")
	if x.r.Intn(3) == 0 {
		w.p("\t%s", x.st.todo(x.r))
	}
	w.p("\treturn nil")
	w.p("}")
	w.p("")
	w.p("// Clone returns a deep copy so callers can mutate the result freely.")
	w.p("func (v *%s) Clone() *%s {", e.Name, e.Name)
	w.p("\tif v == nil {")
	w.p("\t\treturn nil")
	w.p("\t}")
	w.p("\tcp := *v")
	for _, f := range e.Fields {
		if f.Kind == kTags {
			w.p("\tif v.%s != nil {", f.Name)
			w.p("\t\tcp.%s = append([]string(nil), v.%s...)", f.Name, f.Name)
			w.p("\t}")
		}
	}
	w.p("\treturn &cp")
	w.p("}")
	return w.String()
}

func filterField(e Entity) (Field, bool) {
	return firstField(e, func(f Field) bool { return f.Kind == kString && f.Required })
}

func genStore(x *ectx) string {
	e, w, sn, rv := x.e, &writer{}, x.sn, x.st.recv
	w.p("package %s", x.p.Name)
	w.p("")
	w.p("@@IMPORTS@@")
	pl := plural(e.Name)
	ff, hasFF := filterField(e)
	w.p("// %sStore persists %s.", e.Name, strings.ToLower(plural(x.hum)))
	w.p("type %sStore interface {", e.Name)
	w.p("\t%s(ctx context.Context, id string) (*%s, error)", sn.get, e.Name)
	w.p("\t%s(ctx context.Context, v *%s) error", sn.put, e.Name)
	w.p("\t%s(ctx context.Context, id string) error", sn.del)
	w.p("\t%s(ctx context.Context, f %sFilter) ([]*%s, error)", sn.list, e.Name, e.Name)
	w.p("}")
	w.p("")
	w.p("// %sFilter narrows %s results. Zero fields match everything.", e.Name, sn.list)
	w.p("type %sFilter struct {", e.Name)
	w.p("\tStatus %sStatus", e.Name)
	if hasFF {
		w.p("\t%s string", ff.Name)
	}
	w.p("\tLimit int")
	w.p("}")
	w.p("")
	w.p("func (f %sFilter) match(v *%s) bool {", e.Name, e.Name)
	w.p("\tif f.Status != %s && v.Status != f.Status {", x.statusConst("unknown"))
	w.p("\t\treturn false")
	w.p("\t}")
	if hasFF {
		w.p("\tif f.%s != \"\" && v.%s != f.%s {", ff.Name, ff.Name, ff.Name)
		w.p("\t\treturn false")
		w.p("\t}")
	}
	w.p("\treturn true")
	w.p("}")
	w.p("")
	mu := "sync.Mutex"
	rl, ru := "Lock", "Unlock"
	if x.st.rw {
		mu = "sync.RWMutex"
		rl, ru = "RLock", "RUnlock"
	}
	w.p("// Memory%sStore keeps %s in a map. It is safe for concurrent use and is", e.Name, strings.ToLower(plural(x.hum)))
	w.p("// what tests and the local dev server use.")
	w.p("type Memory%sStore struct {", e.Name)
	w.p("\tmu    %s", mu)
	w.p("\titems map[string]*%s", e.Name)
	w.p("}")
	w.p("")
	w.p("// NewMemory%sStore returns an empty store.", e.Name)
	w.p("func NewMemory%sStore() *Memory%sStore {", e.Name, e.Name)
	w.p("\treturn &Memory%sStore{items: make(map[string]*%s)}", e.Name, e.Name)
	w.p("}")
	w.p("")
	w.p("var _ %sStore = (*Memory%sStore)(nil)", e.Name, e.Name)
	w.p("")
	w.p("func (%s *Memory%sStore) %s(ctx context.Context, id string) (*%s, error) {", rv, e.Name, sn.get, e.Name)
	w.p("\tif err := ctx.Err(); err != nil {")
	w.p("\t\treturn nil, err")
	w.p("\t}")
	w.p("\t%s.mu.%s()", rv, rl)
	w.p("\tdefer %s.mu.%s()", rv, ru)
	w.p("\tv, ok := %s.items[id]", rv)
	w.p("\tif !ok {")
	w.p("\t\treturn nil, fmt.Errorf(\"%%w: %s %%q\", ErrNotFound, id)", x.hum)
	w.p("\t}")
	w.p("\treturn v.Clone(), nil")
	w.p("}")
	w.p("")
	w.p("func (%s *Memory%sStore) %s(ctx context.Context, v *%s) error {", rv, e.Name, sn.put, e.Name)
	w.p("\tif err := ctx.Err(); err != nil {")
	w.p("\t\treturn err")
	w.p("\t}")
	w.p("\tif v == nil {")
	w.p("\t\treturn fmt.Errorf(\"%%w: nil %s\", ErrInvalid)", x.hum)
	w.p("\t}")
	w.p("\tif err := v.Validate(); err != nil {")
	w.p("\t\treturn err")
	w.p("\t}")
	w.p("\t%s.mu.Lock()", rv)
	w.p("\tdefer %s.mu.Unlock()", rv)
	w.p("\t%s.items[v.ID] = v.Clone()", rv)
	w.p("\treturn nil")
	w.p("}")
	w.p("")
	w.p("func (%s *Memory%sStore) %s(ctx context.Context, id string) error {", rv, e.Name, sn.del)
	w.p("\tif err := ctx.Err(); err != nil {")
	w.p("\t\treturn err")
	w.p("\t}")
	w.p("\t%s.mu.Lock()", rv)
	w.p("\tdefer %s.mu.Unlock()", rv)
	w.p("\tif _, ok := %s.items[id]; !ok {", rv)
	w.p("\t\treturn fmt.Errorf(\"%%w: %s %%q\", ErrNotFound, id)", x.hum)
	w.p("\t}")
	w.p("\tdelete(%s.items, id)", rv)
	w.p("\treturn nil")
	w.p("}")
	w.p("")
	w.p("// %s returns matching %s ordered by id.", sn.list, strings.ToLower(plural(x.hum)))
	w.p("func (%s *Memory%sStore) %s(ctx context.Context, f %sFilter) ([]*%s, error) {", rv, e.Name, sn.list, e.Name, e.Name)
	w.p("\tif err := ctx.Err(); err != nil {")
	w.p("\t\treturn nil, err")
	w.p("\t}")
	w.p("\t%s.mu.%s()", rv, rl)
	w.p("\tout := make([]*%s, 0, len(%s.items))", e.Name, rv)
	w.p("\tfor _, v := range %s.items {", rv)
	w.p("\t\tif f.match(v) {")
	w.p("\t\t\tout = append(out, v.Clone())")
	w.p("\t\t}")
	w.p("\t}")
	w.p("\t%s.mu.%s()", rv, ru)
	w.p("\tsort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })")
	w.p("\tif f.Limit > 0 && len(out) > f.Limit {")
	w.p("\t\tout = out[:f.Limit]")
	w.p("\t}")
	w.p("\treturn out, nil")
	w.p("}")
	w.p("")
	w.p("// Len reports how many %s are stored.", strings.ToLower(pl))
	w.p("func (%s *Memory%sStore) Len() int {", rv, e.Name)
	w.p("\t%s.mu.%s()", rv, rl)
	w.p("\tdefer %s.mu.%s()", rv, ru)
	w.p("\treturn len(%s.items)", rv)
	w.p("}")
	return w.String()
}

func genCodec(x *ectx) string {
	e, w, h := x.e, &writer{}, x.st.h
	w.p("package %s", x.p.Name)
	w.p("")
	w.p("@@IMPORTS@@")
	w.p("// Encode%s renders v in the key=value wire format used by the nightly", e.Name)
	w.p("// export and the v1 admin screens. Field order is stable.")
	w.p("func Encode%s(v *%s) string {", e.Name, e.Name)
	w.p("\tvar b strings.Builder")
	w.p("\t%s(&b, \"id\", v.ID)", h.kv)
	for _, f := range e.Fields {
		key := snake(f.Name)
		switch f.Kind {
		case kString:
			w.p("\t%s(&b, %q, v.%s)", h.kv, key, f.Name)
		case kInt:
			w.p("\t%s(&b, %q, strconv.Itoa(v.%s))", h.kv, key, f.Name)
		case kInt64:
			w.p("\t%s(&b, %q, strconv.FormatInt(v.%s, 10))", h.kv, key, f.Name)
		case kCents:
			w.p("\t%s(&b, %q, strconv.FormatInt(int64(v.%s), 10))", h.kv, key, f.Name)
		case kBool:
			w.p("\t%s(&b, %q, strconv.FormatBool(v.%s))", h.kv, key, f.Name)
		case kTime:
			w.p("\t%s(&b, %q, %s(v.%s))", h.kv, key, h.ft, f.Name)
		case kTags:
			w.p("\tparts := make([]string, 0, len(v.%s))", f.Name)
			w.p("\tfor _, tag := range v.%s {", f.Name)
			w.p("\t\tparts = append(parts, %s(tag))", h.esc)
			w.p("\t}")
			w.p("\t%s(&b, %q, strings.Join(parts, \",\"))", h.rawkv, key)
		}
	}
	w.p("\t%s(&b, \"status\", v.Status.String())", h.kv)
	w.p("\t%s(&b, \"created_at\", %s(v.CreatedAt))", h.kv, h.ft)
	w.p("\t%s(&b, \"updated_at\", %s(v.UpdatedAt))", h.kv, h.ft)
	w.p("\treturn b.String()")
	w.p("}")
	w.p("")
	w.p("// Decode%s parses the output of Encode%s.", e.Name, e.Name)
	if !x.st.strict {
		w.p("// Unknown keys are ignored so newer exports can be read by older jobs.")
	}
	w.p("func Decode%s(line string) (*%s, error) {", e.Name, e.Name)
	w.p("\tv := &%s{}", e.Name)
	w.p("\tfor _, part := range %s(line, ';') {", h.split)
	w.p("\t\tif part == \"\" {")
	w.p("\t\t\tcontinue")
	w.p("\t\t}")
	w.p("\t\tkey, raw, ok := %s(part, '=')", h.cut)
	w.p("\t\tif !ok {")
	w.p("\t\t\treturn nil, fmt.Errorf(\"%%w: malformed %s field %%q\", ErrInvalid, part)", x.hum)
	w.p("\t\t}")
	w.p("\t\tval := %s(raw)", h.unesc)
	w.p("\t\tswitch %s(key) {", h.unesc)
	w.p("\t\tcase \"id\":")
	w.p("\t\t\tv.ID = val")
	perr := func(key string) {
		w.p("\t\t\tif err != nil {")
		w.p("\t\t\t\treturn nil, fmt.Errorf(\"%%w: %s: %%v\", ErrInvalid, err)", key)
		w.p("\t\t\t}")
	}
	for _, f := range e.Fields {
		key := snake(f.Name)
		w.p("\t\tcase %q:", key)
		switch f.Kind {
		case kString:
			w.p("\t\t\tv.%s = val", f.Name)
		case kInt:
			w.p("\t\t\tn, err := strconv.Atoi(val)")
			perr(key)
			w.p("\t\t\tv.%s = n", f.Name)
		case kInt64:
			w.p("\t\t\tn, err := strconv.ParseInt(val, 10, 64)")
			perr(key)
			w.p("\t\t\tv.%s = n", f.Name)
		case kCents:
			w.p("\t\t\tn, err := strconv.ParseInt(val, 10, 64)")
			perr(key)
			w.p("\t\t\tv.%s = money.Cents(n)", f.Name)
		case kBool:
			w.p("\t\t\tflag, err := strconv.ParseBool(val)")
			perr(key)
			w.p("\t\t\tv.%s = flag", f.Name)
		case kTime:
			w.p("\t\t\tts, err := %s(val)", h.pt)
			perr(key)
			w.p("\t\t\tv.%s = ts", f.Name)
		case kTags:
			w.p("\t\t\tv.%s = nil", f.Name)
			w.p("\t\t\tif raw != \"\" {")
			w.p("\t\t\t\tfor _, tag := range %s(raw, ',') {", h.split)
			w.p("\t\t\t\t\tv.%s = append(v.%s, %s(tag))", f.Name, f.Name, h.unesc)
			w.p("\t\t\t\t}")
			w.p("\t\t\t}")
		}
	}
	w.p("\t\tcase \"status\":")
	w.p("\t\t\tst, err := Parse%sStatus(val)", e.Name)
	w.p("\t\t\tif err != nil {")
	w.p("\t\t\t\treturn nil, err")
	w.p("\t\t\t}")
	w.p("\t\t\tv.Status = st")
	for _, k := range []string{"created_at", "updated_at"} {
		w.p("\t\tcase %q:", k)
		w.p("\t\t\tts, err := %s(val)", h.pt)
		perr(k)
		w.p("\t\t\tv.%s = ts", pascal(k))
	}
	if x.st.strict {
		w.p("\t\tdefault:")
		w.p("\t\t\treturn nil, fmt.Errorf(\"%%w: unknown %s field %%q\", ErrInvalid, key)", x.hum)
	}
	w.p("\t\t}")
	w.p("\t}")
	w.p("\tif v.ID == \"\" {")
	w.p("\t\treturn nil, fmt.Errorf(\"%%w: %s record without id\", ErrInvalid)", x.hum)
	w.p("\t}")
	w.p("\treturn v, nil")
	w.p("}")
	return w.String()
}

type reportFlags struct{ csv, diff bool }

func genReport(x *ectx, rf reportFlags) string {
	e, w := x.e, &writer{}
	w.p("package %s", x.p.Name)
	w.p("")
	pl := plural(e.Name)
	nums := []Field{}
	for _, f := range e.Fields {
		if numeric(f) {
			nums = append(nums, f)
		}
	}
	strf, hasStr := firstField(e, func(f Field) bool { return f.Kind == kString })
	w.p("@@IMPORTS@@")
	w.p("// %sSummary aggregates a batch of %s for the reports package.", e.Name, strings.ToLower(plural(x.hum)))
	w.p("type %sSummary struct {", e.Name)
	w.p("\tCount    int")
	w.p("\tByStatus map[%sStatus]int", e.Name)
	for _, f := range nums {
		t := goType(f)
		if f.Kind == kInt {
			t = "int64"
		}
		w.p("\tTotal%s %s", f.Name, t)
	}
	w.p("\tEarliest time.Time")
	w.p("\tLatest   time.Time")
	w.p("}")
	w.p("")
	w.p("// Summarize%s folds items into a %sSummary. Nil entries are skipped.", pl, e.Name)
	w.p("func Summarize%s(items []*%s) %sSummary {", pl, e.Name, e.Name)
	w.p("\tsum := %sSummary{ByStatus: make(map[%sStatus]int)}", e.Name, e.Name)
	w.p("\tfor _, v := range items {")
	w.p("\t\tif v == nil {")
	w.p("\t\t\tcontinue")
	w.p("\t\t}")
	w.p("\t\tsum.Count++")
	w.p("\t\tsum.ByStatus[v.Status]++")
	for _, f := range nums {
		if f.Kind == kInt {
			w.p("\t\tsum.Total%s += int64(v.%s)", f.Name, f.Name)
		} else {
			w.p("\t\tsum.Total%s += v.%s", f.Name, f.Name)
		}
	}
	w.p("\t\tif sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {")
	w.p("\t\t\tsum.Earliest = v.CreatedAt")
	w.p("\t\t}")
	w.p("\t\tif v.CreatedAt.After(sum.Latest) {")
	w.p("\t\t\tsum.Latest = v.CreatedAt")
	w.p("\t\t}")
	w.p("\t}")
	w.p("\treturn sum")
	w.p("}")
	if len(nums) > 0 {
		f := nums[0]
		w.p("")
		w.p("// Top%sBy%s returns the n items with the largest %s, largest first.", pl, f.Name, words(f.Name))
		w.p("// Ties keep their input order.")
		w.p("func Top%sBy%s(items []*%s, n int) []*%s {", pl, f.Name, e.Name, e.Name)
		w.p("\tout := append([]*%s(nil), items...)", e.Name)
		w.p("\tsort.SliceStable(out, func(i, j int) bool { return out[i].%s > out[j].%s })", f.Name, f.Name)
		w.p("\tif n >= 0 && len(out) > n {")
		w.p("\t\tout = out[:n]")
		w.p("\t}")
		w.p("\treturn out")
		w.p("}")
	}
	if hasStr {
		w.p("")
		w.p("// Group%sBy%s buckets items by %s.", pl, strf.Name, words(strf.Name))
		w.p("func Group%sBy%s(items []*%s) map[string][]*%s {", pl, strf.Name, e.Name, e.Name)
		w.p("\tout := make(map[string][]*%s)", e.Name)
		w.p("\tfor _, v := range items {")
		w.p("\t\tout[v.%s] = append(out[v.%s], v)", strf.Name, strf.Name)
		w.p("\t}")
		w.p("\treturn out")
		w.p("}")
	}
	w.p("")
	w.p("// %sCreatedBetween keeps items created in [from, to).", pl)
	w.p("func %sCreatedBetween(items []*%s, from, to time.Time) []*%s {", pl, e.Name, e.Name)
	w.p("\tvar out []*%s", e.Name)
	w.p("\tfor _, v := range items {")
	w.p("\t\tif !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {")
	w.p("\t\t\tout = append(out, v)")
	w.p("\t\t}")
	w.p("\t}")
	w.p("\treturn out")
	w.p("}")
	if rf.csv {
		w.p("")
		w.p("// Write%sCSV writes items as CSV with a header row.", pl)
		if x.r.Intn(2) == 0 {
			w.p("%s", x.st.todo(x.r))
		}
		w.p("func Write%sCSV(out io.Writer, items []*%s) error {", pl, e.Name)
		w.p("\tcw := csv.NewWriter(out)")
		hdr := []string{"\"id\""}
		for _, f := range e.Fields {
			hdr = append(hdr, fmt.Sprintf("%q", snake(f.Name)))
		}
		hdr = append(hdr, "\"status\"", "\"created_at\"")
		w.p("\tif err := cw.Write([]string{%s}); err != nil {", strings.Join(hdr, ", "))
		w.p("\t\treturn err")
		w.p("\t}")
		w.p("\tfor _, v := range items {")
		w.p("\t\trec := []string{")
		w.p("\t\t\tv.ID,")
		for _, f := range e.Fields {
			switch f.Kind {
			case kString:
				w.p("\t\t\tv.%s,", f.Name)
			case kInt:
				w.p("\t\t\tstrconv.Itoa(v.%s),", f.Name)
			case kInt64:
				w.p("\t\t\tstrconv.FormatInt(v.%s, 10),", f.Name)
			case kCents:
				w.p("\t\t\tmoney.Cents(v.%s).String(),", f.Name)
			case kBool:
				w.p("\t\t\tstrconv.FormatBool(v.%s),", f.Name)
			case kTime:
				w.p("\t\t\t%s(v.%s),", x.st.h.ft, f.Name)
			case kTags:
				w.p("\t\t\tcsvJoin(v.%s),", f.Name)
			}
		}
		w.p("\t\t\tv.Status.String(),")
		w.p("\t\t\t%s(v.CreatedAt),", x.st.h.ft)
		w.p("\t\t}")
		w.p("\t\tif err := cw.Write(rec); err != nil {")
		w.p("\t\t\treturn err")
		w.p("\t\t}")
		w.p("\t}")
		w.p("\tcw.Flush()")
		w.p("\treturn cw.Error()")
		w.p("}")
		if hasKind(e, kTags) {
			w.p("")
			w.p("func csvJoin(vals []string) string {")
			w.p("\tout := \"\"")
			w.p("\tfor i, v := range vals {")
			w.p("\t\tif i > 0 {")
			w.p("\t\t\tout += \"|\"")
			w.p("\t\t}")
			w.p("\t\tout += v")
			w.p("\t}")
			w.p("\treturn out")
			w.p("}")
		}
	}
	if rf.diff {
		w.p("")
		w.p("// Diff%s lists the fields that differ between a and b.", e.Name)
		w.p("func Diff%s(a, b *%s) []FieldChange {", e.Name, e.Name)
		w.p("\tvar out []FieldChange")
		for _, f := range e.Fields {
			key := snake(f.Name)
			switch f.Kind {
			case kTime:
				w.p("\tif !a.%s.Equal(b.%s) {", f.Name, f.Name)
				w.p("\t\tout = append(out, FieldChange{%q, %s(a.%s), %s(b.%s)})", key, x.st.h.ft, f.Name, x.st.h.ft, f.Name)
				w.p("\t}")
			case kTags:
				w.p("\tif strings.Join(a.%s, \",\") != strings.Join(b.%s, \",\") {", f.Name, f.Name)
				w.p("\t\tout = append(out, FieldChange{%q, strings.Join(a.%s, \",\"), strings.Join(b.%s, \",\")})", key, f.Name, f.Name)
				w.p("\t}")
			default:
				w.p("\tif a.%s != b.%s {", f.Name, f.Name)
				w.p("\t\tout = append(out, FieldChange{%q, fmt.Sprint(a.%s), fmt.Sprint(b.%s)})", key, f.Name, f.Name)
				w.p("\t}")
			}
		}
		w.p("\tif a.Status != b.Status {")
		w.p("\t\tout = append(out, FieldChange{\"status\", a.Status.String(), b.Status.String()})")
		w.p("\t}")
		w.p("\treturn out")
		w.p("}")
	}
	return w.String()
}

// ---- tests ----

type sampleInfo struct {
	mul, add map[string]int
}

func genTest(x *ectx, rf reportFlags) string {
	e, w, sn := x.e, &writer{}, x.sn
	w.p("package %s", x.p.Name)
	w.p("")
	w.p("@@IMPORTS@@")
	si := sampleInfo{mul: map[string]int{}, add: map[string]int{}}
	w.p("func sample%s(n int) *%s {", e.Name, e.Name)
	w.p("\treturn &%s{", e.Name)
	w.p("\t\tID: fmt.Sprintf(\"%s-%%03d\", n),", snake(e.Name))
	for k, f := range e.Fields {
		switch f.Kind {
		case kString:
			if k%2 == 0 {
				w.p("\t\t%s: fmt.Sprintf(\"%s %%d;a=b,c\\\\d\", n),", f.Name, strings.ToLower(f.Name))
			} else {
				w.p("\t\t%s: fmt.Sprintf(\"%s-%%d\", n),", f.Name, strings.ToLower(f.Name))
			}
		case kInt, kInt64, kCents:
			m, a := 1+x.r.Intn(40), x.r.Intn(500)
			si.mul[f.Name], si.add[f.Name] = m, a
			cast := ""
			switch f.Kind {
			case kInt64:
				cast = "int64"
			case kCents:
				cast = "money.Cents"
			}
			if cast == "" {
				w.p("\t\t%s: n*%d + %d,", f.Name, m, a)
			} else {
				w.p("\t\t%s: %s(n*%d + %d),", f.Name, cast, m, a)
			}
		case kBool:
			w.p("\t\t%s: n%%2 == 0,", f.Name)
		case kTime:
			w.p("\t\t%s: time.Date(2023, time.March, n, 9, 30, 0, 0, time.UTC),", f.Name)
		case kTags:
			w.p("\t\t%s: []string{\"x,y\", fmt.Sprintf(\"t%%d\", n)},", f.Name)
		}
	}
	w.p("\t\tStatus: %s,", x.statusConst(e.Statuses[0]))
	w.p("\t\tCreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),")
	w.p("\t}")
	w.p("}")

	tests := []func(){
		func() { // codec
			w.p("")
			w.p("func Test%sCodecRoundTrip(t *testing.T) {", e.Name)
			w.p("\tin := sample%s(4)", e.Name)
			w.p("\tline := Encode%s(in)", e.Name)
			w.p("\tout, err := Decode%s(line)", e.Name)
			w.p("\tif err != nil {")
			w.p("\t\tt.Fatalf(\"Decode%s(%%q): %%v\", line, err)", e.Name)
			w.p("\t}")
			w.p("\tif got := Encode%s(out); got != line {", e.Name)
			w.p("\t\tt.Fatalf(\"round trip mismatch:\\n got %%s\\nwant %%s\", got, line)")
			w.p("\t}")
			for _, f := range e.Fields {
				if f.Kind == kString {
					w.p("\tif out.%s != in.%s {", f.Name, f.Name)
					w.p("\t\tt.Errorf(\"%s = %%q, want %%q\", out.%s, in.%s)", f.Name, f.Name, f.Name)
					w.p("\t}")
				}
			}
			w.p("\tif _, err := Decode%s(\"id\"); !errors.Is(err, ErrInvalid) {", e.Name)
			w.p("\t\tt.Fatalf(\"malformed input: err = %%v, want ErrInvalid\", err)")
			w.p("\t}")
			w.p("}")
		},
		func() { // store
			w.p("")
			w.p("func TestMemory%sStore(t *testing.T) {", e.Name)
			w.p("\tctx := context.Background()")
			w.p("\tst := NewMemory%sStore()", e.Name)
			w.p("\tfor n := 1; n <= 3; n++ {")
			w.p("\t\tif err := st.%s(ctx, sample%s(n)); err != nil {", sn.put, e.Name)
			w.p("\t\t\tt.Fatalf(\"%s(%%d): %%v\", n, err)", sn.put)
			w.p("\t\t}")
			w.p("\t}")
			w.p("\tgot, err := st.%s(ctx, \"%s-002\")", sn.get, snake(e.Name))
			w.p("\tif err != nil {")
			w.p("\t\tt.Fatal(err)")
			w.p("\t}")
			w.p("\tgot.Status = %s", x.statusConst(e.Statuses[len(e.Statuses)-1]))
			w.p("\tagain, _ := st.%s(ctx, \"%s-002\")", sn.get, snake(e.Name))
			w.p("\tif again.Status != %s {", x.statusConst(e.Statuses[0]))
			w.p("\t\tt.Fatal(\"store handed out a shared pointer\")")
			w.p("\t}")
			w.p("\tif err := st.%s(ctx, got); err != nil {", sn.put)
			w.p("\t\tt.Fatal(err)")
			w.p("\t}")
			w.p("\tlist, err := st.%s(ctx, %sFilter{Status: %s})", sn.list, e.Name, x.statusConst(e.Statuses[0]))
			w.p("\tif err != nil || len(list) != 2 || list[0].ID != \"%s-001\" {", snake(e.Name))
			w.p("\t\tt.Fatalf(\"%s by status = %%v, %%v\", list, err)", sn.list)
			w.p("\t}")
			w.p("\tif err := st.%s(ctx, \"%s-001\"); err != nil {", sn.del, snake(e.Name))
			w.p("\t\tt.Fatal(err)")
			w.p("\t}")
			w.p("\tif _, err := st.%s(ctx, \"%s-001\"); !errors.Is(err, ErrNotFound) {", sn.get, snake(e.Name))
			w.p("\t\tt.Fatalf(\"after delete: err = %%v, want ErrNotFound\", err)")
			w.p("\t}")
			w.p("\tif st.Len() != 2 {")
			w.p("\t\tt.Fatalf(\"Len = %%d, want 2\", st.Len())")
			w.p("\t}")
			w.p("}")
		},
		func() { // validate
			w.p("")
			w.p("func Test%sValidate(t *testing.T) {", e.Name)
			w.p("\tif err := sample%s(1).Validate(); err != nil {", e.Name)
			w.p("\t\tt.Fatalf(\"valid sample rejected: %%v\", err)")
			w.p("\t}")
			w.p("\tbad := sample%s(1)", e.Name)
			w.p("\tbad.ID = \" \"")
			w.p("\tif err := bad.Validate(); !errors.Is(err, ErrInvalid) {")
			w.p("\t\tt.Fatalf(\"blank id: err = %%v\", err)")
			w.p("\t}")
			if f, ok := filterField(e); ok {
				w.p("\tbad = sample%s(1)", e.Name)
				w.p("\tbad.%s = \"\"", f.Name)
				w.p("\tif err := bad.Validate(); !errors.Is(err, ErrInvalid) {")
				w.p("\t\tt.Fatalf(\"missing %s: err = %%v\", err)", words(f.Name))
				w.p("\t}")
			}
			w.p("\tbad = sample%s(1)", e.Name)
			w.p("\tbad.Status = %s", x.statusConst("unknown"))
			w.p("\tif err := bad.Validate(); err == nil {")
			w.p("\t\tt.Fatal(\"unknown status accepted\")")
			w.p("\t}")
			w.p("}")
		},
		func() { // status
			w.p("")
			w.p("func Test%sStatusTransitions(t *testing.T) {", e.Name)
			w.p("\tfor _, s := range []%sStatus{%s} {", e.Name, x.statusConst(e.Statuses[0])+", "+x.statusConst(e.Statuses[len(e.Statuses)-1]))
			w.p("\t\tgot, err := Parse%sStatus(s.String())", e.Name)
			w.p("\t\tif err != nil || got != s {")
			w.p("\t\t\tt.Fatalf(\"Parse(%%q) = %%v, %%v\", s.String(), got, err)")
			w.p("\t\t}")
			w.p("\t}")
			w.p("\tif !%s.CanTransition(%s) {", x.statusConst(e.Statuses[0]), x.statusConst(e.Statuses[1]))
			w.p("\t\tt.Fatal(\"first transition should be allowed\")")
			w.p("\t}")
			w.p("\tif %s.CanTransition(%s) {", x.statusConst(e.Statuses[1]), x.statusConst(e.Statuses[0]))
			w.p("\t\tt.Fatal(\"backwards transition allowed\")")
			w.p("\t}")
			w.p("\tif !%s.Terminal() {", x.statusConst(e.Statuses[len(e.Statuses)-1]))
			w.p("\t\tt.Fatal(\"last status should be terminal\")")
			w.p("\t}")
			w.p("}")
		},
		func() { // summary
			nums := []Field{}
			for _, f := range e.Fields {
				if numeric(f) {
					nums = append(nums, f)
				}
			}
			w.p("")
			w.p("func TestSummarize%s(t *testing.T) {", plural(e.Name))
			w.p("\titems := []*%s{sample%s(1), nil, sample%s(2), sample%s(3)}", e.Name, e.Name, e.Name, e.Name)
			w.p("\titems[2].Status = %s", x.statusConst(e.Statuses[1]))
			w.p("\tsum := Summarize%s(items)", plural(e.Name))
			w.p("\tif sum.Count != 3 || sum.ByStatus[%s] != 2 || sum.ByStatus[%s] != 1 {", x.statusConst(e.Statuses[0]), x.statusConst(e.Statuses[1]))
			w.p("\t\tt.Fatalf(\"counts = %%+v\", sum)")
			w.p("\t}")
			for _, f := range nums {
				want := si.mul[f.Name]*6 + si.add[f.Name]*3
				w.p("\tif sum.Total%s != %d {", f.Name, want)
				w.p("\t\tt.Errorf(\"Total%s = %%v, want %d\", sum.Total%s)", f.Name, want, f.Name)
				w.p("\t}")
			}
			w.p("\tif !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {")
			w.p("\t\tt.Errorf(\"range = %%v..%%v\", sum.Earliest, sum.Latest)")
			w.p("\t}")
			if len(nums) > 0 {
				w.p("\tif top := Top%sBy%s(items[2:], 1); len(top) != 1 || top[0].ID != \"%s-003\" {", plural(e.Name), nums[0].Name, snake(e.Name))
				w.p("\t\tt.Errorf(\"top = %%v\", top)")
				w.p("\t}")
			}
			w.p("}")
		},
	}
	order := x.r.Perm(len(tests))
	n := 2 + x.r.Intn(3)
	chosen := append([]int(nil), order[:n]...)
	sort.Ints(chosen)
	for _, k := range chosen {
		tests[k]()
	}
	if rf.csv && x.r.Intn(2) == 0 {
		w.p("")
		w.p("func TestWrite%sCSV(t *testing.T) {", plural(e.Name))
		w.p("\tvar buf bytes.Buffer")
		w.p("\tif err := Write%sCSV(&buf, []*%s{sample%s(1), sample%s(2)}); err != nil {", plural(e.Name), e.Name, e.Name, e.Name)
		w.p("\t\tt.Fatal(err)")
		w.p("\t}")
		w.p("\tlines := strings.Split(strings.TrimSpace(buf.String()), \"\\n\")")
		w.p("\tif len(lines) != 3 || !strings.HasPrefix(lines[0], \"id,\") {")
		w.p("\t\tt.Fatalf(\"csv = %%q\", buf.String())")
		w.p("\t}")
		w.p("}")
	}
	return w.String()
}
