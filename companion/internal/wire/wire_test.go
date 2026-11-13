package wire

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLimitsWorstCaseUnder128KiB(t *testing.T) {
	var req SyncRequest
	req.V = 1
	str := strings.Repeat("x", MaxFieldBytes)
	for i := 0; i < MaxCharacters; i++ {
		c := Character{Name: str, Realm: str, Class: str, Race: str, Level: 60, Seen: 1763000000}
		for j := 0; j < MaxSkills; j++ {
			c.Skills = append(c.Skills, Skill{ID: 999999, Rank: 300, Max: 300})
		}
		req.Characters = append(req.Characters, c)
	}
	b, _ := json.Marshal(req)
	if len(b) >= MaxBodyBytes {
		t.Fatalf("worst case %d bytes does not fit %d", len(b), MaxBodyBytes)
	}
}

func TestRequestBodyMatchesVector(t *testing.T) {
	b, _ := json.Marshal(SyncRequest{V: 1})
	if string(b) != `{"v":1,"game_running":false}` {
		t.Fatal(string(b))
	}
}

func TestParseRosterSkipsOutOfBounds(t *testing.T) {
	long := strings.Repeat("a", MaxMemberName+1)
	var chars []string
	for i := 0; i < MaxMemberChars+5; i++ {
		chars = append(chars, `{"name":"C`+strings.Repeat("x", 3)+`"}`)
	}
	raw := `{"members":[{"name":"` + long + `","characters":[]},{"name":"Anna","characters":[` +
		`{"name":"` + strings.Repeat("b", 65) + `"},{"name":"Thrallina","class":"Shaman","level":23,"main":true},` +
		`{"name":"Lvl","level":23.5}]},{"name":"Many","characters":[` + strings.Join(chars, ",") + `]},{"name":5}]}`
	r, err := ParseRoster([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Members) != 2 || r.Members[0].Name != "Anna" || len(r.Members[0].Characters) != 2 {
		t.Fatalf("%+v", r)
	}
	if r.Members[0].Characters[1].Level != 0 || !r.Members[0].Characters[0].Main {
		t.Fatalf("%+v", r.Members[0])
	}
	if len(r.Members[1].Characters) != MaxMemberChars {
		t.Fatal(len(r.Members[1].Characters))
	}
}

func TestParseRosterWrongShape(t *testing.T) {
	for _, raw := range []string{`[]`, `{}`, `{"members":{}}`, `"x"`, ``} {
		if _, err := ParseRoster([]byte(raw)); err == nil {
			t.Errorf("%q accepted", raw)
		}
	}
}
