#!/usr/bin/env python3
"""Turn a WoW SavedVariables file into JSON and a short summary.

A SavedVariables file is plain Lua: one or more top-level assignments of the form
``Name = { ... }`` with nested tables, strings, numbers, booleans and nil. The client writes
it on logout and ``/reload``. This parser handles exactly that dialect (no expressions, no
functions), so it stays small and needs no Lua on the machine that reads the data.

Usage: parse_sv.py <file.lua> [--json out.json] [--quiet]
"""
import argparse
import json
import re
import sys

TOKEN = re.compile(
    r"""\s*(?:
        (?P<comment>--[^\n]*) |
        (?P<lstr>\[(=*)\[.*?\]\3\]) |
        (?P<str>"(?:\\.|[^"\\])*") |
        (?P<num>-?(?:0[xX][0-9a-fA-F]+|\d+\.?\d*(?:[eE][-+]?\d+)?|\.\d+(?:[eE][-+]?\d+)?)) |
        (?P<name>[A-Za-z_][A-Za-z0-9_]*) |
        (?P<punct>[{}\[\]=,;])
    )""",
    re.S | re.X,
)
ESCAPES = {"n": "\n", "t": "\t", "r": "\r", "\\": "\\", '"': '"', "'": "'", "\n": "\n"}


def tokens(text):
    pos = 0
    while pos < len(text):
        m = TOKEN.match(text, pos)
        if not m:
            if text[pos:].strip() == "":
                return
            raise SyntaxError(f"unexpected input at offset {pos}: {text[pos:pos + 20]!r}")
        pos = m.end()
        kind = m.lastgroup
        if kind == "comment" or kind is None:
            continue
        yield kind, m.group(kind)


def unescape(s):
    body = s[1:-1]
    out, i = [], 0
    while i < len(body):
        c = body[i]
        if c == "\\" and i + 1 < len(body):
            nxt = body[i + 1]
            if nxt.isdigit():
                j = i + 1
                while j < len(body) and j < i + 4 and body[j].isdigit():
                    j += 1
                out.append(chr(int(body[i + 1:j])))
                i = j
                continue
            out.append(ESCAPES.get(nxt, nxt))
            i += 2
        else:
            out.append(c)
            i += 1
    return "".join(out)


class Parser:
    def __init__(self, text):
        self.toks = list(tokens(text))
        self.i = 0

    def peek(self):
        return self.toks[self.i] if self.i < len(self.toks) else (None, None)

    def take(self, kind=None, value=None):
        k, v = self.peek()
        if k is None or (kind and k != kind) or (value is not None and v != value):
            raise SyntaxError(f"expected {value or kind}, got {v!r} at token {self.i}")
        self.i += 1
        return v

    def file(self):
        out = {}
        while self.peek()[0] is not None:
            name = self.take("name")
            self.take("punct", "=")
            out[name] = self.value()
            if self.peek() == ("punct", ";"):
                self.i += 1
        return out

    def value(self):
        k, v = self.peek()
        if k == "punct" and v == "{":
            return self.table()
        self.i += 1
        if k == "str":
            return unescape(v)
        if k == "lstr":
            return re.sub(r"^\[=*\[\n?|\]=*\]$", "", v)
        if k == "num":
            return int(v, 16) if v.lower().startswith(("0x", "-0x")) else (int(v) if re.fullmatch(r"-?\d+", v) else float(v))
        if k == "name":
            return {"true": True, "false": False, "nil": None}[v] if v in ("true", "false", "nil") else v
        raise SyntaxError(f"unexpected {v!r}")

    def table(self):
        self.take("punct", "{")
        items, array, n = {}, True, 0
        while self.peek() != ("punct", "}"):
            k, v = self.peek()
            if k == "punct" and v == "[":
                self.i += 1
                key = self.value()
                self.take("punct", "]")
                self.take("punct", "=")
                val = self.value()
                array = False
            elif k == "name" and self.toks[self.i + 1] == ("punct", "="):
                key = self.take("name")
                self.take("punct", "=")
                val = self.value()
                array = False
            else:
                n += 1
                key, val = n, self.value()
            items[key] = val
            if self.peek() in (("punct", ","), ("punct", ";")):
                self.i += 1
        self.take("punct", "}")
        if array and items:
            return [items[i] for i in range(1, n + 1)]
        # JSON keys must be strings; keep numeric keys readable.
        return {str(k): v for k, v in items.items()}


def summarise(data, lines):
    for name, value in data.items():
        lines.append(f"{name}: {describe(value)}")
        if isinstance(value, dict):
            for k, v in list(value.items())[:12]:
                lines.append(f"  {k}: {describe(v)}")
            if len(value) > 12:
                lines.append(f"  ... {len(value) - 12} more keys")


def describe(v):
    if isinstance(v, dict):
        return f"table with {len(v)} keys"
    if isinstance(v, list):
        return f"list of {len(v)}"
    if isinstance(v, str):
        return repr(v if len(v) < 60 else v[:57] + "...")
    return repr(v)


def main(argv):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("file")
    ap.add_argument("--json", help="write the parsed data here")
    ap.add_argument("--quiet", action="store_true", help="no summary on stdout")
    a = ap.parse_args(argv)
    with open(a.file, encoding="utf-8", errors="replace") as f:
        data = Parser(f.read()).file()
    if a.json:
        with open(a.json, "w", encoding="utf-8") as f:
            json.dump(data, f, indent=1, ensure_ascii=False)
    if not a.quiet:
        lines = []
        summarise(data, lines)
        print("\n".join(lines))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
