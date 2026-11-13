package dataaddon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

func lua(t *testing.T) string {
	p, err := exec.LookPath("lua5.1")
	if err != nil {
		t.Skip("lua5.1 not on PATH")
	}
	return p
}

// loadCheck loads a Data.lua under an empty environment and prints what it holds; globals other
// than ForeverFriendsData must not appear.
const loadCheck = `
local env = {}
local f = assert(loadfile(arg[1]))
setfenv(f, env)
f()
local n = 0
for k in pairs(env) do n = n + 1 end
assert(n == 1 and type(env.ForeverFriendsData) == "table", "unexpected globals")
local m = env.ForeverFriendsData.members
local m1 = m[1]
io.write(m1.name, "\n", m1.characters[1].name, "\n", #m)
`

func TestEscapeEveryByte0to255(t *testing.T) {
	luaBin := lua(t)
	var all []byte
	for i := 0; i < 256; i++ {
		all = append(all, byte(i), '1') // a digit after each byte: \1 then "1" must stay two bytes
	}
	dir := t.TempDir()
	src := "ForeverFriendsData = { members = { { name = " + Quote(string(all)) + " } } }\n"
	p := filepath.Join(dir, "Data.lua")
	_ = os.WriteFile(p, []byte(src), 0o644)
	script := filepath.Join(dir, "check.lua")
	_ = os.WriteFile(script, []byte(`
local env = {}
local f = assert(loadfile(arg[1])); setfenv(f, env); f()
local s = env.ForeverFriendsData.members[1].name
assert(#s == 512, #s)
for i = 0, 255 do assert(s:byte(2*i+1) == i and s:byte(2*i+2) == 49, i) end
io.write("ok")`), 0o644)
	out, err := exec.Command(luaBin, script, p).CombinedOutput()
	if err != nil || string(out) != "ok" {
		t.Fatalf("%v %s", err, out)
	}
}

func TestHostileStringsStayLiterals(t *testing.T) {
	luaBin := lua(t)
	hostile := []string{`"]]..os.execute("x")..[[`, `\`, "\\1" + "2", "line\nbreak", "nul\x00byte", "del\x7f", `"}, os.exit(1), {"`, "a]]b", `--[[`, "Zoë ü 名前"}
	for _, h := range hostile {
		dir := t.TempDir()
		r := wire.Roster{Members: []wire.RosterMember{{Name: h, Characters: []wire.RosterChar{{Name: h, Class: h, Level: 3, Main: true}}}}}
		if _, err := Write(dir, r, "16001", time.Unix(5, 0)); err != nil {
			t.Fatal(err)
		}
		script := filepath.Join(dir, "check.lua")
		_ = os.WriteFile(script, []byte(loadCheck), 0o644)
		out, err := exec.Command(luaBin, script, filepath.Join(dir, "Data.lua")).CombinedOutput()
		want := h + "\n" + h + "\n1"
		if err != nil || string(out) != want {
			t.Fatalf("%q: %v %q", h, err, out)
		}
	}
}

func TestQuoteForm(t *testing.T) {
	if got := Quote("a\\b\"c\x01\x7fé"); got != `"a\\b\"c\001\127é"` {
		t.Fatal(got)
	}
}

func TestRenderNumbersOnlyFromFormatter(t *testing.T) {
	r := wire.Roster{Members: []wire.RosterMember{{Name: "Anna", Characters: []wire.RosterChar{{Name: "T", Class: "Shaman", Level: 23, Main: true}, {Name: "B", Class: "Mage"}}}}}
	got := string(Render(r, 1763064000))
	for _, want := range []string{"written = 1763064000,", `{ name = "T", class = "Shaman", level = 23, main = true },`, `{ name = "B", class = "Mage" },`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if !strings.Contains(string(Render(wire.Roster{}, 0)), "members = {},") {
		t.Fatal("empty roster")
	}
}

func TestWriteOnlyWhenDifferentIgnoringWritten(t *testing.T) {
	dir := t.TempDir()
	r := wire.Roster{Members: []wire.RosterMember{{Name: "Anna"}}}
	if ch, err := Write(dir, r, "16001", time.Unix(100, 0)); err != nil || !ch {
		t.Fatal(ch, err)
	}
	if ch, err := Write(dir, r, "16001", time.Unix(999, 0)); err != nil || ch {
		t.Fatal("rewritten for a new clock only", ch, err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "Data.lua"))
	if !strings.Contains(string(b), "written = 100,") {
		t.Fatal(string(b))
	}
	r.Members[0].Name = "Berta"
	if ch, _ := Write(dir, r, "16001", time.Unix(200, 0)); !ch {
		t.Fatal("change not written")
	}
	b, _ = os.ReadFile(filepath.Join(dir, "Data.lua"))
	if !strings.Contains(string(b), "written = 200,") || !strings.Contains(string(b), "Berta") {
		t.Fatal(string(b))
	}
}

func TestTocInterfaceCopiedAndValidated(t *testing.T) {
	b, err := Toc("16001")
	if err != nil || !strings.HasPrefix(string(b), "## Interface: 16001\n") || !strings.HasSuffix(string(b), "Data.lua\n") {
		t.Fatal(string(b), err)
	}
	for _, bad := range []string{"", "16001\n## Evil: 1", "abc", "1"} {
		if _, err := Toc(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestEnsureEmptyKeepsExistingList(t *testing.T) {
	dir := filepath.Join(t.TempDir(), FolderName)
	if err := EnsureEmpty(dir, "16001"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "Data.lua"))
	if string(b) != string(Render(wire.Roster{}, 0)) {
		t.Fatal(string(b))
	}
	_, _ = Write(dir, wire.Roster{Members: []wire.RosterMember{{Name: "Anna"}}}, "16001", time.Unix(9, 0))
	if err := EnsureEmpty(dir, "16001"); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "Data.lua"))
	if !strings.Contains(string(b), "Anna") {
		t.Fatal("list wiped")
	}
}
