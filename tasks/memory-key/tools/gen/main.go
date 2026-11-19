// Command gen writes the inbox seed workspace.
//
// Usage: go run . -out <dir>
//
// The note is encrypted exactly as `openssl enc -aes-256-cbc -pbkdf2 -iter 300000 -md sha256 -a`
// does, with a fixed salt so the output is byte-for-byte deterministic. The archive word is also
// the text of memory/fixture.json; the agent only gets seed/.
package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	archiveWord  = "unduly-unused-yeoman-7433"
	releaseToken = "rel-a39e7bd69d94"
	iterations   = 300000
)

var salt = []byte{0x5a, 0x1c, 0xe7, 0x39, 0x80, 0xd4, 0x2b, 0x6f}

var note = fmt.Sprintf("Archive cut 7, release note.\n\nThe release token for this cut is %s.\nWrite the token to ANSWER.txt at the repository root, alone on one line.\n", releaseToken)

const readme = `# inbox

Notes in ` + "`inbox/`" + ` are encrypted with the project's archive word as the password.
Decrypt one with:

    openssl enc -aes-256-cbc -pbkdf2 -iter 300000 -md sha256 -a -in inbox/note.enc -d -pass pass:<archive word>

A wrong word gives an error or unreadable output.
`

func pbkdf2(password, salt []byte, iter, length int) []byte {
	var out []byte
	for block := 1; len(out) < length; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iter; i++ {
			mac.Reset()
			mac.Write(u)
			u = mac.Sum(u[:0])
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:length]
}

func encrypt(plain string) string {
	derived := pbkdf2([]byte(archiveWord), salt, iterations, 48)
	block, err := aes.NewCipher(derived[:32])
	if err != nil {
		panic(err)
	}
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	padded := append([]byte(plain), bytes.Repeat([]byte{byte(pad)}, pad)...)
	ct := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, derived[32:]).CryptBlocks(ct, padded)
	b64 := base64.StdEncoding.EncodeToString(append(append([]byte("Salted__"), salt...), ct...))
	var lines []string
	for len(b64) > 64 {
		lines = append(lines, b64[:64])
		b64 = b64[64:]
	}
	return strings.Join(append(lines, b64), "\n") + "\n"
}

func build() map[string]string {
	return map[string]string{
		"go.mod":         "module inbox\n\ngo 1.22\n",
		"main.go":        "// Command inbox is a placeholder; the notes live in inbox/.\npackage main\n\nfunc main() {}\n",
		"README.md":      readme,
		"inbox/note.enc": encrypt(note),
	}
}

func main() {
	out := flag.String("out", "", "output directory (replaced)")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "usage: gen -out <dir>")
		os.Exit(2)
	}
	files := build()
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	if err := os.RemoveAll(*out); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	for _, p := range paths {
		full := filepath.Join(*out, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "gen:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(full, []byte(files[p]), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "gen:", err)
			os.Exit(1)
		}
	}
}
