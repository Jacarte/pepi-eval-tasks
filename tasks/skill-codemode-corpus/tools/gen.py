#!/usr/bin/env python3
"""Generate seed/corpus/ for the skill-codemode-corpus task.

Usage: tools/gen.py [OUT_DIR]   (default: ../seed/corpus next to tools/)

Deterministic: the same script always writes the same bytes (fixed seed, no clock, no
dict-order dependence). Every run removes OUT_DIR first. Run `tools/gen.py` and
`git status` to see whether the committed corpus is current.

Layout produced: 1000 counted files (.txt, .md, .go) in directories whose names start with
"_" (so `go build ./...` skips the .go files, which are text, not code), plus a few ignored
files (other extensions, uppercase .TXT) under data/.
Edge cases: CRLF line endings, no final newline, comment-only files, an empty file,
indented comments, non-ASCII letters, whitespace-only lines, and block comments.
"""
import os
import random
import shutil
import sys

SEED = 20241118
FILE_COUNT = 1000

DIRS = ["_alpha", "_beta", "_gamma", "_notes/q1", "_notes/q2", "_src/core", "_src/util", "_docs/deep/er"]
STEMS = ["note", "entry", "chunk", "memo", "item", "log", "draft"]
WORDS = (
    "the a of to and in is it for on that with as was at by this from be or an are not "
    "we you they have had build test value parser cache queue retry timeout server client "
    "request response buffer stream token index record column table schema migrate "
    "release notes review merge branch commit rollback latency budget metric trace span "
    "alpha beta gamma delta orchid ledger harbor lantern compass velvet quartz nimbus"
).split()
APOS = ["don't", "it's", "can't", "o'clock", "rock'n'roll", "'quoted'", "won't"]
UNI = ["café", "naïve", "résumé", "Straße", "ñandú", "Ωmega", "привет", "日本語", "façade", "coöperate"]
PUNCT = ["42", "3.14", "x+y", "--flag", "(see", "[1]", "a/b", "#1", "v2.0", "e.g."]


def prose(rng, uni):
    n = rng.randint(3, 14)
    out = []
    for _ in range(n):
        r = rng.random()
        if uni and r < 0.25:
            out.append(rng.choice(UNI))
        elif r < 0.10:
            out.append(rng.choice(APOS))
        elif r < 0.16:
            out.append(rng.choice(PUNCT))
        else:
            out.append(rng.choice(WORDS))
    text = " ".join(out)
    return text[0].upper() + text[1:] + rng.choice([".", ",", "", "!", "?"])


def code_line(rng, uni):
    r = rng.random()
    if r < 0.2:
        return "\treturn " + prose(rng, uni).lower().rstrip(".,!?") + "(" + str(rng.randint(0, 999)) + ")"
    if r < 0.4:
        return "func %s%d(n int) int { return n * %d }" % (rng.choice(WORDS), rng.randint(0, 99), rng.randint(1, 9))
    if r < 0.55:
        return "\tx%d := %d + len(\"%s\")" % (rng.randint(0, 9), rng.randint(0, 99), rng.choice(WORDS))
    if r < 0.65:
        return "}"
    if r < 0.7:
        return "/* " + prose(rng, uni) + " */"
    return ""


def comment_line(rng, ext, uni, indent):
    body = prose(rng, uni)
    if ext == ".go":
        line = ("//" if rng.random() < 0.6 else "// ") + body
    else:
        line = ("#" if rng.random() < 0.6 else "# ") + body
    return indent + line


def make_lines(rng, ext, n, mode):
    uni = rng.random() < (0.5 if mode == "unicode" else 0.12)
    lines = []
    for _ in range(n):
        r = rng.random()
        if mode == "comments":
            lines.append("" if r < 0.1 else comment_line(rng, ext, uni, ""))
        elif mode == "indented":
            if r < 0.5:
                lines.append(comment_line(rng, ext, uni, rng.choice(["  ", "\t", "    \t"])))
            else:
                lines.append(prose(rng, uni))
        elif r < 0.12:
            lines.append(rng.choice(["", "   ", "\t"]))
        elif r < 0.22:
            lines.append(comment_line(rng, ext, uni, ""))
        elif ext == ".go" and r < 0.5:
            lines.append(code_line(rng, uni))
        elif ext != ".go" and r < 0.27:
            lines.append(prose(rng, uni) + " # not a comment")
        else:
            lines.append(prose(rng, uni))
    return lines


def file_bytes(rng, i, ext):
    if i == 0:
        return b""  # empty file: zero lines
    if i % 113 == 11:
        mode = "comments"
    elif i % 41 == 3:
        mode = "indented"
    elif i % 61 == 9:
        mode = "unicode"
    else:
        mode = "normal"
    n = rng.randint(1, 60)
    lines = make_lines(rng, ext, n, mode)
    eol = "\r\n" if i % 97 == 5 else "\n"
    text = eol.join(lines)
    if i % 89 != 7:  # every file except the no-final-newline ones ends with a newline
        text += eol
    return text.encode("utf-8")


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    out = sys.argv[1] if len(sys.argv) > 1 else os.path.join(here, "..", "seed", "corpus")
    out = os.path.abspath(out)
    if os.path.exists(out):
        shutil.rmtree(out)
    rng = random.Random(SEED)
    used = set()
    for i in range(FILE_COUNT):
        ext = rng.choice([".txt", ".md", ".go"])
        if ext == ".go":
            d = rng.choice(DIRS)
        else:
            d = rng.choice(DIRS + ["", "", ""])
        name = "%s_%04d%s" % (rng.choice(STEMS), i, ext)
        rel = os.path.join(d, name) if d else name
        if rel in used:
            raise SystemExit("duplicate path " + rel)
        used.add(rel)
        path = os.path.join(out, rel)
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, "wb") as f:
            f.write(file_bytes(rng, i, ext))
    ignored = {
        "data/config.json": b'{"name": "corpus", "count": 1000}\n',
        "data/NOTES.TXT": b"uppercase extension: not counted\n",
        "data/draft.txt.bak": b"backup copy: not counted\n",
        "data/image.md.png": b"\x89PNG not text\n",
    }
    for rel, content in ignored.items():
        path = os.path.join(out, rel)
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, "wb") as f:
            f.write(content)


if __name__ == "__main__":
    main()
