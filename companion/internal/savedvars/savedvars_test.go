package savedvars

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

// pythonShape renders a Value the way tools/parse_sv.py does: a table whose keys are 1..n in
// order is a list, any other a map with string keys.
func pythonShape(v Value) any {
	t, ok := v.(*Table)
	if !ok {
		return v
	}
	list := true
	for i, k := range t.Keys {
		if f, ok := k.(float64); !ok || f != float64(i+1) {
			list = false
		}
	}
	if list && len(t.Keys) > 0 {
		out := make([]any, len(t.Vals))
		for i, x := range t.Vals {
			out[i] = pythonShape(x)
		}
		return out
	}
	m := map[string]any{}
	for i, k := range t.Keys {
		var ks string
		switch kk := k.(type) {
		case float64:
			ks = strconv.FormatFloat(kk, 'g', -1, 64)
		case string:
			ks = kk
		default:
			ks = fmt.Sprint(kk)
		}
		m[ks] = pythonShape(t.Vals[i])
	}
	return m
}

func normalise(t *testing.T, v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestParseAgreesWithPythonGolden(t *testing.T) {
	src, _ := os.ReadFile("testdata/sample.lua")
	got, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	shaped := map[string]any{}
	for k, v := range got {
		shaped[k] = pythonShape(v)
	}
	golden, _ := os.ReadFile("testdata/golden.json")
	var want any
	if err := json.Unmarshal(golden, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(normalise(t, shaped), want) {
		t.Fatalf("Go parser and golden differ:\n%v\n%v", normalise(t, shaped), want)
	}
}

func TestParseAgreesWithPythonLive(t *testing.T) {
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("no python3")
	}
	out, err := exec.Command(py, "-I", "../../../tools/parse_sv.py", "testdata/sample.lua", "--json", "/dev/stdout", "--quiet").Output()
	if err != nil {
		t.Skip("parse_sv.py could not run here:", err)
	}
	golden, _ := os.ReadFile("testdata/golden.json")
	var a, b any
	if json.Unmarshal(out, &a) != nil || json.Unmarshal(golden, &b) != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("golden.json is stale: regenerate it with tools/parse_sv.py")
	}
}

func TestEscapesAndLongStrings(t *testing.T) {
	top, err := Parse([]byte("X = { \"a\\\"b\\\\c\\n\\065\\0661\\a\", [==[\nx]]y]==], 'q' }\n"))
	if err != nil {
		t.Fatal(err)
	}
	l := top["X"].(*Table).List()
	if l[0] != "a\"b\\c\nAB1\a" || l[1] != "x]]y" || l[2] != "q" {
		t.Fatalf("%q", l)
	}
}

func TestMalformedNeverPanics(t *testing.T) {
	for _, s := range []string{"", "X", "X =", "X = {", "X = { a =", "X = \"abc", "X = { [1 = 2 }", "X = [[", "X = 1e", "X = {{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{", "X = \\", "X = \"\\999\"", "= 5", "X = foo", "-- [[ never closed", "X = 0x"} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%q panicked: %v", s, r)
				}
			}()
			_, _ = Parse([]byte(s))
		}()
	}
}

func TestEveryPrefixOfSampleIsAnErrorOrWhole(t *testing.T) {
	src, _ := os.ReadFile("testdata/sample.lua")
	text := string(src)
	cut := strings.LastIndex(text[:strings.Index(text, "OtherVar")], "}")
	for i := 0; i < cut; i++ {
		if _, err := Parse(src[:i]); err == nil && i > 0 && strings.TrimSpace(text[:i]) != "" {
			t.Fatalf("a file cut at %d parsed", i)
		}
	}
}

func FuzzParse(f *testing.F) {
	src, _ := os.ReadFile("testdata/sample.lua")
	f.Add(src)
	f.Add([]byte("X = { [1] = 2 }"))
	f.Fuzz(func(t *testing.T, b []byte) { _, _ = Parse(b) })
}

func fsWith(files map[string]string, mtime time.Time) fstest.MapFS {
	m := fstest.MapFS{}
	for p, c := range files {
		m[p] = &fstest.MapFile{Data: []byte(c), ModTime: mtime}
	}
	return m
}

func account(name string) string { return "WTF/Account/" + name + "/SavedVariables/ForeverFriends.lua" }

func db(body string) string {
	return "ForeverFriendsDB = { version = 1, characters = { " + body + " } }"
}

func ch(name, realm string, seen int, extra string) string {
	return fmt.Sprintf(`["%s-%s"] = { name = "%s", realm = "%s", class = "SHAMAN", level = 20, seen = %d, %s },`, name, realm, name, realm, seen, extra)
}

func TestMergeNewerSeenWins(t *testing.T) {
	fsys := fsWith(map[string]string{
		account("A"): db(ch("Thrall", "R", 100, `race = "Orc"`) + ch("OnlyA", "R", 50, "")),
		account("B"): db(ch("Thrall", "R", 200, `race = "Troll"`)),
	}, time.Unix(1, 0))
	s := Read(fsys)
	if len(s.Files) != 2 || !s.Files[0].OK || !s.Files[1].OK {
		t.Fatalf("%+v", s.Files)
	}
	if len(s.Characters) != 2 || s.Characters[0].Race != "Troll" || s.Characters[1].Name != "OnlyA" {
		t.Fatalf("%+v", s.Characters)
	}
}

func TestUploadCaps(t *testing.T) {
	var body strings.Builder
	for i := 0; i < 51; i++ {
		fmt.Fprint(&body, ch(fmt.Sprintf("C%02d", i), "R", 1000+i, ""))
	}
	body.WriteString(ch("TooLong", strings.Repeat("r", 65), 9999, ""))
	var sk strings.Builder
	for i := 0; i < 41; i++ {
		fmt.Fprintf(&sk, `{ id = %d, name = "n", rank = 1, max = 75 },`, 100+i)
	}
	body.WriteString(ch("Skilled", "R", 5000, "skills = { "+sk.String()+" }"))
	s := Read(fsWith(map[string]string{account("A"): db(body.String())}, time.Unix(1, 0)))
	if len(s.Characters) != wire.MaxCharacters {
		t.Fatal(len(s.Characters))
	}
	if s.Characters[0].Name != "Skilled" || len(s.Characters[0].Skills) != wire.MaxSkills {
		t.Fatalf("%+v", s.Characters[0])
	}
	for _, c := range s.Characters {
		if c.Name == "TooLong" {
			t.Fatal("65-byte realm sent")
		}
	}
	b, _ := json.Marshal(s.Characters[0])
	if strings.Contains(string(b), `"name":"n"`) {
		t.Fatal("skill name sent")
	}
}

func TestPartialAndForeignFiles(t *testing.T) {
	fsys := fsWith(map[string]string{
		account("A"): "ForeverFriendsDB = { version = 1, characters = { [\"X-R\"] = { name = ",
		account("B"): "ForeverFriendsDB = { version = 2, characters = {} }",
		account("C"): db(ch("Good", "R", 5, "")),
		account("D"): db(`["noclass"] = { name = "N", realm = "R" }, ["notable"] = 3,`),
	}, time.Unix(1, 0))
	s := Read(fsys)
	ok := map[string]bool{}
	for _, f := range s.Files {
		ok[f.Path] = f.OK
	}
	if ok[account("A")] || ok[account("B")] || !ok[account("C")] || !ok[account("D")] {
		t.Fatalf("%+v", s.Files)
	}
	if len(s.Characters) != 1 || s.Characters[0].Name != "Good" {
		t.Fatalf("%+v", s.Characters)
	}
}

func TestNoFilesIsEmpty(t *testing.T) {
	s := Read(fstest.MapFS{})
	if len(s.Files) != 0 || len(s.Characters) != 0 {
		t.Fatal(s)
	}
}

func TestFingerprintChangesWithFile(t *testing.T) {
	a := Read(fsWith(map[string]string{account("A"): db("")}, time.Unix(1, 0)))
	b := Read(fsWith(map[string]string{account("A"): db("")}, time.Unix(2, 0)))
	if a.Files[0].Fingerprint == b.Files[0].Fingerprint {
		t.Fatal("same fingerprint")
	}
}

// The client runs on Windows and may write CRLF; git on Windows may also check the sample out
// that way. Line endings between tokens, after a comment and at the end of the file must not
// matter, and a line break inside a long string reads as one \n, as in Lua.
func TestCRLFReadsLikeLF(t *testing.T) {
	src, _ := os.ReadFile("testdata/sample.lua")
	lf := strings.ReplaceAll(string(src), "\r\n", "\n")
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")
	a, err := Parse([]byte(lf))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Parse([]byte(crlf))
	if err != nil {
		t.Fatalf("CRLF form: %v", err)
	}
	if !reflect.DeepEqual(normalise(t, pythonShape(&Table{Keys: []Value{"a"}, Vals: []Value{tableOf(a)}})), normalise(t, pythonShape(&Table{Keys: []Value{"a"}, Vals: []Value{tableOf(b)}}))) {
		t.Fatal("CRLF file parsed differently")
	}
	for _, s := range []string{"X = 1 -- comment\r\n", "X = { 1, -- [1]\r\n 2 } -- end", "X = \"a\\\r\nb\"\r\n"} {
		if _, err := Parse([]byte(s)); err != nil {
			t.Errorf("%q: %v", s, err)
		}
	}
	top, _ := Parse([]byte("X = \"a\\\r\nb\"\r\n"))
	if top["X"] != "a\nb" {
		t.Fatalf("%q", top["X"])
	}
}

func tableOf(m map[string]Value) *Table {
	t := &Table{}
	for k, v := range m {
		t.Keys, t.Vals = append(t.Keys, k), append(t.Vals, v)
	}
	return t
}

// The hub replaces a character's professions with a `skills` that is present, even an empty one,
// so an entry without usable skill lines must leave the field out of the body.
func TestEntryWithoutSkillsSendsNoSkillsField(t *testing.T) {
	for name, extra := range map[string]string{"no key": "", "empty table": "skills = {}", "no usable line": "skills = { { name = \"x\" }, \"junk\" }"} {
		s := Read(fsWith(map[string]string{account("A"): db(ch("Thrall", "R", 5, extra))}, time.Unix(1, 0)))
		if len(s.Characters) != 1 {
			t.Fatalf("%s: %+v", name, s)
		}
		b, _ := json.Marshal(s.Characters[0])
		if strings.Contains(string(b), "skills") || strings.Contains(string(b), "null") {
			t.Errorf("%s: %s", name, b)
		}
	}
	s := Read(fsWith(map[string]string{account("A"): db(ch("Thrall", "R", 5, "skills = { { id = 171, rank = 1, max = 75 } }"))}, time.Unix(1, 0)))
	if b, _ := json.Marshal(s.Characters[0]); !strings.Contains(string(b), `"skills":[{"id":171`) {
		t.Fatalf("%s", b)
	}
}
