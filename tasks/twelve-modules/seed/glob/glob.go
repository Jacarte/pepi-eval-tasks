// Package glob implements gitignore-style path matching. See SPEC.md section 2.
package glob

// Match reports whether path is ignored by the gitignore-style patterns.
func Match(patterns []string, path string, isDir bool) bool { return false }
