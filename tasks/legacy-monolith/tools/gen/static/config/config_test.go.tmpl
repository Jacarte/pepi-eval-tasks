package config

import (
	"errors"
	"strings"
	"testing"

	"shopcore/errs"
)

func TestDefaultsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range defaults {
		if seen[s.Key] {
			t.Errorf("duplicate key %s", s.Key)
		}
		seen[s.Key] = true
		if !strings.Contains(s.Key, ".") {
			t.Errorf("key %s has no section", s.Key)
		}
	}
	if len(Defaults().Keys()) != len(defaults) {
		t.Fatal("Keys() lost entries")
	}
}

func TestLoadOverrides(t *testing.T) {
	var key string
	for _, s := range defaults {
		if s.Kind == "int" {
			key = s.Key
			break
		}
	}
	c, err := Load(strings.NewReader("# comment\n\n" + key + " = 4242\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Int(key, -1) != 4242 {
		t.Fatalf("%s = %d", key, c.Int(key, -1))
	}
	_, err = Load(strings.NewReader("nope.nothing = 1\n"))
	if !errors.Is(err, ErrUnknownKey) || errs.Op(err) != "config.load" {
		t.Fatalf("err = %v", err)
	}
	_, err = Load(strings.NewReader(key + " = many\n"))
	if !errors.Is(err, ErrBadValue) {
		t.Fatalf("err = %v", err)
	}
}
