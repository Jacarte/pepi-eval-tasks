package semver_test

import (
	"reflect"
	"strings"
	"testing"

	"fivemods/semver"
)

func TestM1ParseValid(t *testing.T) {
	cases := []struct {
		in                  string
		major, minor, patch uint64
		pre, build          []string
	}{
		{"0.0.0", 0, 0, 0, nil, nil},
		{"1.2.3", 1, 2, 3, nil, nil},
		{"10.20.30", 10, 20, 30, nil, nil},
		{"1.0.0-alpha", 1, 0, 0, []string{"alpha"}, nil},
		{"1.0.0-alpha.1", 1, 0, 0, []string{"alpha", "1"}, nil},
		{"1.0.0-0.3.7", 1, 0, 0, []string{"0", "3", "7"}, nil},
		{"1.0.0-x.7.z.92", 1, 0, 0, []string{"x", "7", "z", "92"}, nil},
		{"1.0.0-x-y-z.--", 1, 0, 0, []string{"x-y-z", "--"}, nil},
		{"1.0.0+20130313144700", 1, 0, 0, nil, []string{"20130313144700"}},
		{"1.0.0-beta+exp.sha.5114f85", 1, 0, 0, []string{"beta"}, []string{"exp", "sha", "5114f85"}},
		{"1.0.0+21AF26D3----117B344092BD", 1, 0, 0, nil, []string{"21AF26D3----117B344092BD"}},
		{"1.0.0+001.0002", 1, 0, 0, nil, []string{"001", "0002"}},
		{"1.0.0-0alpha", 1, 0, 0, []string{"0alpha"}, nil},
		{"18446744073709551615.0.1", 18446744073709551615, 0, 1, nil, nil},
		{"1.2.3-99999999999999999999999", 1, 2, 3, []string{"99999999999999999999999"}, nil},
		{"2.0.0-rc.1+build.123", 2, 0, 0, []string{"rc", "1"}, []string{"build", "123"}},
	}
	for _, c := range cases {
		v, err := semver.Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", c.in, err)
			continue
		}
		if v.Major != c.major || v.Minor != c.minor || v.Patch != c.patch {
			t.Errorf("Parse(%q) = %d.%d.%d, want %d.%d.%d", c.in, v.Major, v.Minor, v.Patch, c.major, c.minor, c.patch)
		}
		if !reflect.DeepEqual(v.Prerelease, c.pre) {
			t.Errorf("Parse(%q).Prerelease = %#v, want %#v", c.in, v.Prerelease, c.pre)
		}
		if !reflect.DeepEqual(v.Build, c.build) {
			t.Errorf("Parse(%q).Build = %#v, want %#v", c.in, v.Build, c.build)
		}
		if got := v.String(); got != c.in {
			t.Errorf("Parse(%q).String() = %q", c.in, got)
		}
	}
}

func TestM1ParseInvalid(t *testing.T) {
	cases := []string{
		"", "1", "1.2", "1.2.3.4", "v1.2.3", " 1.2.3", "1.2.3 ", "01.2.3", "1.02.3", "1.2.03",
		"1.2.3-", "1.2.3+", "1.2.3-01", "1.2.3-alpha..1", "1.2.3-alpha.", "1.2.3+build..1",
		"1.2.3-alpha_1", "1.2.3+build+x", "1.2.3-α", "-1.2.3", "1.-2.3", "1.2.-3", "a.b.c",
		"18446744073709551616.0.0", "1.2.3-alpha.01", "1..3", "1.2.3-+b", "+1.2.3",
	}
	for _, in := range cases {
		if v, err := semver.Parse(in); err == nil {
			t.Errorf("Parse(%q) = %v, want error", in, v)
		}
	}
	defer func() {
		if recover() == nil {
			t.Errorf("MustParse(\"1.2\") did not panic")
		}
	}()
	semver.MustParse("1.2")
}

func TestM1Compare(t *testing.T) {
	chain := []string{
		"0.0.1", "0.1.0", "0.9.0", "0.10.0", "1.0.0-0", "1.0.0-1", "1.0.0-2", "1.0.0-10",
		"1.0.0-99999999999999999998", "1.0.0-99999999999999999999", "1.0.0-1a",
		"1.0.0-ALPHA", "1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.1.0", "1.0.0-alpha.beta",
		"1.0.0-alpha-1", "1.0.0-beta", "1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0",
		"1.0.1-alpha", "1.0.1", "1.1.0", "1.9.9", "1.10.0", "2.0.0", "10.0.0",
	}
	for i := range chain {
		for j := range chain {
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = 1
			}
			a, b := semver.MustParse(chain[i]), semver.MustParse(chain[j])
			if got := semver.Compare(a, b); got != want {
				t.Errorf("Compare(%s, %s) = %d, want %d", chain[i], chain[j], got, want)
			}
		}
	}
	equal := [][2]string{
		{"1.0.0+a", "1.0.0+b"}, {"1.0.0-rc.1+x", "1.0.0-rc.1"}, {"1.2.3+001", "1.2.3+1"},
	}
	for _, e := range equal {
		if got := semver.Compare(semver.MustParse(e[0]), semver.MustParse(e[1])); got != 0 {
			t.Errorf("Compare(%s, %s) = %d, want 0", e[0], e[1], got)
		}
	}
}

func TestM1Sort(t *testing.T) {
	in := []string{"1.0.0+b2", "1.0.0-rc.1", "0.9.9", "1.0.0+b1", "1.0.0-alpha", "1.0.0", "1.0.0-alpha.10", "1.0.0-alpha.9"}
	want := []string{"0.9.9", "1.0.0-alpha", "1.0.0-alpha.9", "1.0.0-alpha.10", "1.0.0-rc.1", "1.0.0+b2", "1.0.0+b1", "1.0.0"}
	vs := make([]semver.Version, len(in))
	for i, s := range in {
		vs[i] = semver.MustParse(s)
	}
	semver.Sort(vs)
	got := make([]string, len(vs))
	for i, v := range vs {
		got[i] = v.String()
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Sort = %v, want %v", got, want)
	}

	// Many equal-precedence versions: order among equals must be preserved.
	var big []semver.Version
	for i := 0; i < 60; i++ {
		big = append(big, semver.Version{Major: 1, Minor: uint64((i * 7) % 4), Build: []string{"b" + strings.Repeat("x", i)}})
	}
	semver.Sort(big)
	for i := 1; i < len(big); i++ {
		a, b := big[i-1], big[i]
		if a.Minor > b.Minor || (a.Minor == b.Minor && len(a.Build[0]) > len(b.Build[0])) {
			t.Fatalf("Sort not stable/ordered at %d: %s before %s", i, a, b)
		}
	}
}

func TestM1Increment(t *testing.T) {
	cases := []struct {
		in, op, id, want string
	}{
		{"1.2.3", "major", "", "2.0.0"},
		{"1.2.3", "minor", "", "1.3.0"},
		{"1.2.3", "patch", "", "1.2.4"},
		{"1.2.3+build.7", "patch", "", "1.2.4"},
		{"1.0.0-alpha", "major", "", "1.0.0"},
		{"1.1.0-alpha", "major", "", "2.0.0"},
		{"1.0.1-alpha", "major", "", "2.0.0"},
		{"1.2.0-rc.1", "minor", "", "1.2.0"},
		{"1.2.1-rc.1", "minor", "", "1.3.0"},
		{"1.2.3-rc.1", "patch", "", "1.2.3"},
		{"0.0.9", "patch", "", "0.0.10"},
		{"1.2.3", "pre", "", "1.2.4-0"},
		{"1.2.3", "pre", "rc", "1.2.4-rc.0"},
		{"1.2.3-alpha.5", "pre", "beta", "1.2.3-beta.0"},
		{"1.2.3-alpha.5", "pre", "alpha", "1.2.3-alpha.6"},
		{"1.2.3-alpha.5", "pre", "", "1.2.3-alpha.6"},
		{"1.0.0-rc.1.x", "pre", "", "1.0.0-rc.2.x"},
		{"1.0.0-alpha", "pre", "", "1.0.0-alpha.0"},
		{"1.0.0-alpha", "pre", "alpha", "1.0.0-alpha.0"},
		{"1.0.0-9", "pre", "", "1.0.0-10"},
		{"1.0.0-1.2.99", "pre", "", "1.0.0-1.2.100"},
		{"1.0.0-99999999999999999999", "pre", "", "1.0.0-100000000000000000000"},
		{"1.0.0-beta.1+meta", "pre", "beta", "1.0.0-beta.2"},
	}
	for _, c := range cases {
		v := semver.MustParse(c.in)
		var got semver.Version
		switch c.op {
		case "major":
			got = v.IncMajor()
		case "minor":
			got = v.IncMinor()
		case "patch":
			got = v.IncPatch()
		case "pre":
			got = v.IncPrerelease(c.id)
		}
		if got.String() != c.want {
			t.Errorf("%s.Inc(%s,%q) = %s, want %s", c.in, c.op, c.id, got, c.want)
		}
		if got.Build != nil {
			t.Errorf("%s.Inc(%s) kept build %v", c.in, c.op, got.Build)
		}
		if v.String() != c.in {
			t.Errorf("Inc mutated receiver: %s became %s", c.in, v)
		}
	}
}

func TestM1Range(t *testing.T) {
	cases := []struct {
		rng string
		in  []string
		out []string
	}{
		{">=1.2.0 <2.0.0 || ^3.1.0", []string{"1.2.0", "1.9.9", "3.1.0", "3.99.0"}, []string{"1.1.9", "2.0.0", "3.0.9", "4.0.0", "2.5.0"}},
		{"1.2.3", []string{"1.2.3", "1.2.3+build"}, []string{"1.2.4", "1.2.2", "1.2.3-rc.1"}},
		{"=1.2.3", []string{"1.2.3"}, []string{"1.2.4"}},
		{">1.2.3", []string{"1.2.4", "2.0.0"}, []string{"1.2.3", "1.0.0", "1.2.4-alpha"}},
		{"<1.2.3", []string{"1.2.2", "0.0.1"}, []string{"1.2.3", "1.2.3-alpha", "2.0.0"}},
		{"<=1.2.3", []string{"1.2.3", "1.0.0"}, []string{"1.2.4"}},
		{"^1.2.3", []string{"1.2.3", "1.9.0", "1.99.99"}, []string{"1.2.2", "2.0.0", "2.0.0-alpha", "1.5.0-rc.1"}},
		{"^0.2.3", []string{"0.2.3", "0.2.99"}, []string{"0.3.0", "0.2.2", "1.0.0"}},
		{"^0.0.3", []string{"0.0.3"}, []string{"0.0.4", "0.0.2", "0.1.0"}},
		{"^0.0.0", []string{"0.0.0"}, []string{"0.0.1"}},
		{"~1.2.3", []string{"1.2.3", "1.2.99"}, []string{"1.3.0", "1.2.2", "2.0.0"}},
		{"~0.0.5", []string{"0.0.5", "0.0.9"}, []string{"0.1.0", "0.0.4"}},
		{"^1.2.3-beta.1", []string{"1.2.3-beta.1", "1.2.3-beta.2", "1.2.3", "1.2.5"}, []string{"1.2.3-alpha", "1.2.4-alpha", "1.2.3-beta.0", "2.0.0"}},
		{"<2.0.0-beta", []string{"1.9.0", "2.0.0-alpha"}, []string{"2.0.0-beta", "2.0.0-rc", "1.9.0-alpha"}},
		{"1.2.3 - 2.3.4", []string{"1.2.3", "2.0.0", "2.3.4"}, []string{"1.2.2", "2.3.5", "2.3.4-rc.1"}},
		{"1.0.0-rc.1 - 1.0.0", []string{"1.0.0-rc.1", "1.0.0-rc.2", "1.0.0"}, []string{"1.0.0-beta", "1.0.1"}},
		{"*", []string{"0.0.0", "99.0.0", "1.2.3+b"}, []string{"1.0.0-rc.1"}},
		{"  >=1.0.0\t<1.1.0   ||   >=3.0.0  ", []string{"1.0.5", "3.2.1"}, []string{"1.1.0", "2.0.0"}},
		{">=1.0.0 <1.0.0", nil, []string{"1.0.0", "0.9.0"}},
		{"1.0.0 || 2.0.0 || 3.0.0-rc.1", []string{"1.0.0", "2.0.0", "3.0.0-rc.1"}, []string{"3.0.0", "1.5.0", "3.0.0-rc.2"}},
		{">1.0.0-alpha <1.0.0", []string{"1.0.0-alpha.1", "1.0.0-beta"}, []string{"1.0.0-alpha", "1.0.0", "0.9.0"}},
		{">=1.0.0 * <2.0.0", []string{"1.5.0"}, []string{"2.0.0", "1.5.0-rc.1"}},
		{"* || 1.0.0-alpha", []string{"1.0.0-alpha", "5.0.0"}, []string{"1.0.0-beta", "2.0.0-alpha"}},
		{"~1.2.3-rc.1", []string{"1.2.3-rc.2", "1.2.9"}, []string{"1.2.4-rc.1", "1.3.0"}},
		{">=0.0.0", []string{"0.0.0", "1.0.0"}, []string{"0.0.0-alpha", "1.0.0-alpha"}},
	}
	for _, c := range cases {
		r, err := semver.ParseRange(c.rng)
		if err != nil {
			t.Errorf("ParseRange(%q) error: %v", c.rng, err)
			continue
		}
		for _, s := range c.in {
			if !r.Contains(semver.MustParse(s)) {
				t.Errorf("%q should contain %s", c.rng, s)
			}
		}
		for _, s := range c.out {
			if r.Contains(semver.MustParse(s)) {
				t.Errorf("%q should not contain %s", c.rng, s)
			}
		}
	}
}

func TestM1RangeInvalid(t *testing.T) {
	cases := []string{
		"", "   ", "1.2.3 ||", "|| 1.2.3", "1.2.3 || || 2.0.0", ">= 1.2.3", ">=", "1.2", "x",
		"~>1.2.3", ">=1.0.0 - 2.0.0", "1.0.0 - >=2.0.0", "1.0.0 -", "- 1.0.0", "^1.2", "1.2.3 abc",
		">>1.2.3", "<=v1.2.3", "1.0.0 - 2.0",
	}
	for _, in := range cases {
		if _, err := semver.ParseRange(in); err == nil {
			t.Errorf("ParseRange(%q) succeeded, want error", in)
		}
	}
}

func TestM1MaxSatisfying(t *testing.T) {
	parse := func(list string) []semver.Version {
		var vs []semver.Version
		for _, s := range strings.Fields(list) {
			vs = append(vs, semver.MustParse(s))
		}
		return vs
	}
	cases := []struct {
		vs, rng, want string
		ok            bool
	}{
		{"1.2.3 1.2.4 1.3.0 2.0.0", "^1.2.0", "1.3.0", true},
		{"1.2.3 1.2.4 1.3.0 2.0.0", "~1.2.0", "1.2.4", true},
		{"1.0.0 2.0.0-rc.1 1.5.0", "<3.0.0", "1.5.0", true},
		{"2.0.0-rc.1 2.0.0-rc.2 1.0.0", ">=2.0.0-rc.1", "2.0.0-rc.2", true},
		{"1.0.0+first 0.5.0 1.0.0+second", "*", "1.0.0+first", true},
		{"0.1.0 0.2.0", ">=1.0.0", "0.0.0", false},
		{"", "*", "0.0.0", false},
	}
	for _, c := range cases {
		r, err := semver.ParseRange(c.rng)
		if err != nil {
			t.Fatalf("ParseRange(%q): %v", c.rng, err)
		}
		got, ok := semver.MaxSatisfying(parse(c.vs), r)
		if ok != c.ok || got.String() != c.want {
			t.Errorf("MaxSatisfying(%q, %q) = %s,%v want %s,%v", c.vs, c.rng, got, ok, c.want, c.ok)
		}
	}
}
