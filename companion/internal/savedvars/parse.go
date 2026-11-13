// Package savedvars reads the file the game client writes for the addon (docs/CONTRACT.md §1).
// The file is Lua, but only the dialect the client emits: top-level assignments of tables,
// strings, numbers, booleans and nil. Nothing is executed.
package savedvars

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Limits that keep a hostile or damaged file from costing more than it is worth.
const (
	maxDepth = 32
	// MaxFileBytes is the largest file the companion reads.
	MaxFileBytes = 8 << 20
)

// Value is nil, bool, float64, string or *Table.
type Value any

// Table is a Lua table in source order; positional items have the float64 index as key.
type Table struct {
	Keys []Value
	Vals []Value
}

// Get returns the value under a string key, or nil.
func (t *Table) Get(name string) Value {
	for i, k := range t.Keys {
		if s, ok := k.(string); ok && s == name {
			return t.Vals[i]
		}
	}
	return nil
}

// List returns the values under the keys 1, 2, 3 ... in order, stopping at the first gap.
func (t *Table) List() []Value {
	byIndex := map[int]Value{}
	for i, k := range t.Keys {
		if f, ok := k.(float64); ok && f == float64(int(f)) && f >= 1 {
			byIndex[int(f)] = t.Vals[i]
		}
	}
	var out []Value
	for n := 1; ; n++ {
		v, ok := byIndex[n]
		if !ok {
			return out
		}
		out = append(out, v)
	}
}

// ErrSyntax is returned for anything outside the dialect, including a file cut off in the middle.
var ErrSyntax = errors.New("not a SavedVariables file")

type parser struct {
	s     string
	i     int
	depth int
}

func fail(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrSyntax, fmt.Sprintf(format, a...))
}

// Parse reads a whole file into its top-level assignments.
func Parse(src []byte) (map[string]Value, error) {
	p := &parser{s: string(src)}
	out := map[string]Value{}
	for {
		if err := p.skip(); err != nil {
			return nil, err
		}
		if p.i >= len(p.s) {
			return out, nil
		}
		name, ok := p.ident()
		if !ok {
			return nil, fail("expected a name at offset %d", p.i)
		}
		if err := p.skip(); err != nil {
			return nil, err
		}
		if !p.eat('=') {
			return nil, fail("expected = after %s", name)
		}
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		out[name] = v
		if err := p.skip(); err != nil {
			return nil, err
		}
		p.eat(';')
	}
}

func (p *parser) eat(c byte) bool {
	if p.i < len(p.s) && p.s[p.i] == c {
		p.i++
		return true
	}
	return false
}

// skip passes whitespace and comments, line and long.
func (p *parser) skip() error {
	for p.i < len(p.s) {
		c := p.s[p.i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			p.i++
		case strings.HasPrefix(p.s[p.i:], "--"):
			p.i += 2
			if lvl, ok := p.longOpen(); ok {
				if _, err := p.longBody(lvl); err != nil {
					return err
				}
				continue
			}
			for p.i < len(p.s) && p.s[p.i] != '\n' {
				p.i++
			}
		default:
			return nil
		}
	}
	return nil
}

func isIdentStart(c byte) bool { return c == '_' || (c|0x20 >= 'a' && c|0x20 <= 'z') }
func isDigit(c byte) bool      { return c >= '0' && c <= '9' }

func (p *parser) ident() (string, bool) {
	start := p.i
	if p.i >= len(p.s) || !isIdentStart(p.s[p.i]) {
		return "", false
	}
	for p.i < len(p.s) && (isIdentStart(p.s[p.i]) || isDigit(p.s[p.i])) {
		p.i++
	}
	return p.s[start:p.i], true
}

// longOpen consumes [=*[ and returns the number of equals signs.
func (p *parser) longOpen() (int, bool) {
	j := p.i
	if j >= len(p.s) || p.s[j] != '[' {
		return 0, false
	}
	j++
	lvl := 0
	for j < len(p.s) && p.s[j] == '=' {
		lvl++
		j++
	}
	if j < len(p.s) && p.s[j] == '[' {
		p.i = j + 1
		return lvl, true
	}
	return 0, false
}

func (p *parser) longBody(lvl int) (string, error) {
	closer := "]" + strings.Repeat("=", lvl) + "]"
	end := strings.Index(p.s[p.i:], closer)
	if end < 0 {
		return "", fail("long string not closed")
	}
	body := p.s[p.i : p.i+end]
	p.i += end + len(closer)
	if strings.HasPrefix(body, "\r\n") {
		body = body[2:]
	} else if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}
	return body, nil
}

func (p *parser) value() (Value, error) {
	if err := p.skip(); err != nil {
		return nil, err
	}
	if p.i >= len(p.s) {
		return nil, fail("file ends where a value was expected")
	}
	c := p.s[p.i]
	switch {
	case c == '{':
		return p.table()
	case c == '"' || c == '\'':
		return p.quoted(c)
	case c == '[':
		if lvl, ok := p.longOpen(); ok {
			return p.longBody(lvl)
		}
		return nil, fail("unexpected [ at offset %d", p.i)
	case c == '-' || c == '.' || isDigit(c):
		return p.number()
	case isIdentStart(c):
		name, _ := p.ident()
		switch name {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "nil":
			return nil, nil
		}
		return nil, fail("unexpected name %q", name)
	}
	return nil, fail("unexpected %q at offset %d", c, p.i)
}

func (p *parser) number() (Value, error) {
	start := p.i
	if p.s[p.i] == '-' {
		p.i++
	}
	if strings.HasPrefix(p.s[p.i:], "0x") || strings.HasPrefix(p.s[p.i:], "0X") {
		p.i += 2
		for p.i < len(p.s) && strings.ContainsRune("0123456789abcdefABCDEF", rune(p.s[p.i])) {
			p.i++
		}
	} else {
		for p.i < len(p.s) && (isDigit(p.s[p.i]) || p.s[p.i] == '.') {
			p.i++
		}
		if p.i < len(p.s) && (p.s[p.i] == 'e' || p.s[p.i] == 'E') {
			p.i++
			if p.i < len(p.s) && (p.s[p.i] == '+' || p.s[p.i] == '-') {
				p.i++
			}
			for p.i < len(p.s) && isDigit(p.s[p.i]) {
				p.i++
			}
		}
	}
	text := p.s[start:p.i]
	neg := strings.HasPrefix(text, "-")
	body := strings.TrimPrefix(text, "-")
	if strings.HasPrefix(body, "0x") || strings.HasPrefix(body, "0X") {
		n, err := strconv.ParseUint(body[2:], 16, 64)
		if err != nil {
			return nil, fail("bad number %q", text)
		}
		f := float64(n)
		if neg {
			f = -f
		}
		return f, nil
	}
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, fail("bad number %q", text)
	}
	return f, nil
}

var simpleEscapes = map[byte]byte{'n': '\n', 't': '\t', 'r': '\r', 'a': 7, 'b': 8, 'f': 12, 'v': 11, '\\': '\\', '"': '"', '\'': '\'', '\n': '\n'}

func (p *parser) quoted(q byte) (Value, error) {
	p.i++
	var b []byte
	for p.i < len(p.s) {
		c := p.s[p.i]
		switch {
		case c == q:
			p.i++
			return string(b), nil
		case c == '\n':
			return nil, fail("string not closed")
		case c != '\\':
			b = append(b, c)
			p.i++
		default:
			p.i++
			if p.i >= len(p.s) {
				return nil, fail("string not closed")
			}
			e := p.s[p.i]
			if isDigit(e) {
				n, j := 0, p.i
				for j < len(p.s) && j < p.i+3 && isDigit(p.s[j]) {
					n = n*10 + int(p.s[j]-'0')
					j++
				}
				if n > 255 {
					return nil, fail("escape \\%d out of range", n)
				}
				b = append(b, byte(n))
				p.i = j
				continue
			}
			if r, ok := simpleEscapes[e]; ok {
				b = append(b, r)
			} else {
				b = append(b, e)
			}
			p.i++
		}
	}
	return nil, fail("string not closed")
}

func (p *parser) table() (Value, error) {
	if p.depth >= maxDepth {
		return nil, fail("tables nested deeper than %d", maxDepth)
	}
	p.depth++
	defer func() { p.depth-- }()
	p.i++ // {
	t := &Table{}
	pos := 0
	for {
		if err := p.skip(); err != nil {
			return nil, err
		}
		if p.i >= len(p.s) {
			return nil, fail("file ends inside a table")
		}
		if p.eat('}') {
			return t, nil
		}
		var key Value
		switch c := p.s[p.i]; {
		case c == '[' && !p.isLongString():
			p.i++
			k, err := p.value()
			if err != nil {
				return nil, err
			}
			if err := p.skip(); err != nil {
				return nil, err
			}
			if !p.eat(']') {
				return nil, fail("expected ] at offset %d", p.i)
			}
			key = k
			if err := p.expectEquals(); err != nil {
				return nil, err
			}
		case isIdentStart(c) && p.identThenEquals():
			key, _ = p.ident()
			if err := p.expectEquals(); err != nil {
				return nil, err
			}
		default:
			pos++
			key = float64(pos)
		}
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		t.Keys, t.Vals = append(t.Keys, key), append(t.Vals, v)
		if err := p.skip(); err != nil {
			return nil, err
		}
		if !p.eat(',') {
			p.eat(';')
		}
	}
}

func (p *parser) isLongString() bool {
	save := p.i
	_, ok := p.longOpen()
	p.i = save
	return ok
}

func (p *parser) identThenEquals() bool {
	save := p.i
	defer func() { p.i = save }()
	if _, ok := p.ident(); !ok {
		return false
	}
	if p.skip() != nil {
		return false
	}
	return p.i < len(p.s) && p.s[p.i] == '=' && !strings.HasPrefix(p.s[p.i:], "==")
}

func (p *parser) expectEquals() error {
	if err := p.skip(); err != nil {
		return err
	}
	if !p.eat('=') {
		return fail("expected = at offset %d", p.i)
	}
	return nil
}
