// Package semver parses, compares, increments and range-matches Semantic
// Versioning 2.0.0 versions. See SPEC.md section 1.
package semver

import "errors"

var errNotImplemented = errors.New("semver: not implemented")

// Version is a parsed semantic version.
type Version struct {
	Major, Minor, Patch uint64
	Prerelease          []string // dot-separated pre-release identifiers; nil when absent
	Build               []string // dot-separated build identifiers; nil when absent
}

// Parse parses a strict SemVer 2.0.0 version string.
func Parse(s string) (Version, error) { return Version{}, errNotImplemented }

// MustParse is like Parse but panics if Parse fails.
func MustParse(s string) Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the canonical text of v.
func (v Version) String() string { return "" }

// Compare returns -1, 0 or +1 by SemVer precedence; build metadata is ignored.
func Compare(a, b Version) int { return 0 }

// IncMajor returns the next major version.
func (v Version) IncMajor() Version { return Version{} }

// IncMinor returns the next minor version.
func (v Version) IncMinor() Version { return Version{} }

// IncPatch returns the next patch version.
func (v Version) IncPatch() Version { return Version{} }

// IncPrerelease returns the next pre-release version.
func (v Version) IncPrerelease(id string) Version { return Version{} }

// Sort sorts vs in ascending precedence order (stable).
func Sort(vs []Version) {}

// Range is a parsed version range.
type Range struct{}

// ParseRange parses a range expression such as ">=1.2.0 <2.0.0 || ^3.1.0".
func ParseRange(s string) (Range, error) { return Range{}, errNotImplemented }

// Contains reports whether v satisfies r.
func (r Range) Contains(v Version) bool { return false }

// MaxSatisfying returns the highest version in vs contained in r.
func MaxSatisfying(vs []Version, r Range) (Version, bool) { return Version{}, false }
