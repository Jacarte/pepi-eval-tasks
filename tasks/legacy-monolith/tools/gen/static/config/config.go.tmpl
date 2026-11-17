// Package config loads the flat key=value settings file shipped with every
// deploy. Keys are "<package>.<knob>"; see defaults_table.go for the list.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"shopcore/errs"
)

type setting struct {
	Key   string
	Kind  string // bool, int or string
	Value string
	Doc   string
}

var (
	ErrUnknownKey = errors.New("config: unknown key")
	ErrBadValue   = errors.New("config: bad value")
)

// Config is an immutable-after-load settings snapshot.
type Config struct {
	values map[string]string
	kinds  map[string]string
}

// Defaults returns the built-in settings.
func Defaults() *Config {
	c := &Config{values: make(map[string]string, len(defaults)), kinds: make(map[string]string, len(defaults))}
	for _, s := range defaults {
		c.values[s.Key] = s.Value
		c.kinds[s.Key] = s.Kind
	}
	return c
}

// Load reads overrides from r on top of Defaults. Blank lines and lines
// starting with # are ignored.
func Load(r io.Reader) (*Config, error) {
	const op = "config.load"
	c := Defaults()
	sc := bufio.NewScanner(r)
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		k, v, ok := strings.Cut(text, "=")
		if !ok {
			return nil, errs.Wrap(op, fmt.Errorf("line %d: %w: missing '='", line, ErrBadValue))
		}
		if err := c.Set(strings.TrimSpace(k), strings.TrimSpace(v)); err != nil {
			return nil, errs.Wrap(op, fmt.Errorf("line %d: %w", line, err))
		}
	}
	if err := sc.Err(); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return c, nil
}

// Set overrides one key after checking its kind.
func (c *Config) Set(key, value string) error {
	kind, ok := c.kinds[key]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}
	switch kind {
	case "bool":
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("%w: %s=%q", ErrBadValue, key, value)
		}
	case "int":
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("%w: %s=%q", ErrBadValue, key, value)
		}
	}
	c.values[key] = value
	return nil
}

// String returns the raw value of key, or "" if unknown.
func (c *Config) String(key string) string { return c.values[key] }

// Int returns key as an int. Unknown keys and non-int kinds return fallback.
func (c *Config) Int(key string, fallback int) int {
	if c.kinds[key] != "int" {
		return fallback
	}
	n, err := strconv.Atoi(c.values[key])
	if err != nil {
		return fallback
	}
	return n
}

// Bool returns key as a bool, false when unknown.
func (c *Config) Bool(key string) bool {
	b, _ := strconv.ParseBool(c.values[key])
	return b
}

// Keys lists every known key in sorted order.
func (c *Config) Keys() []string {
	out := make([]string, 0, len(c.values))
	for k := range c.values {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Section returns the keys under prefix ("orders" -> "orders.*").
func (c *Config) Section(prefix string) map[string]string {
	out := map[string]string{}
	for k, v := range c.values {
		if strings.HasPrefix(k, prefix+".") {
			out[strings.TrimPrefix(k, prefix+".")] = v
		}
	}
	return out
}
