// Package wire holds the JSON shapes and numeric limits of docs/CONTRACT.md §3, so that the
// package which sends, the one which reads the game's file and the one which writes the data
// addon cannot disagree about them.
package wire

import (
	"encoding/json"
	"errors"
)

// Limits the contract sets on either side of the wire.
const (
	Version        = 1
	MaxCharacters  = 50
	MaxSkills      = 40
	MaxFieldBytes  = 64
	MaxBodyBytes   = 128 << 10
	MaxMemberName  = 160
	MaxMemberChars = 60
	// MaxDeviceName is the length of a PC's name the hub keeps, in characters.
	MaxDeviceName = 40
	// MaxAnswerBytes bounds what the companion reads of any answer; a member list for fifteen
	// people is a few KiB, so this only stops a runaway server.
	MaxAnswerBytes = 4 << 20
)

// Skill is one skill line as sent: the localized name stays on the PC.
type Skill struct {
	ID   int `json:"id"`
	Rank int `json:"rank"`
	Max  int `json:"max"`
}

// Character is one entry of the upload. Skills is nil, and so left out of the body, unless the
// addon's entry has at least one usable line: the hub replaces a character's professions with
// whatever a present `skills` holds, an empty list included.
type Character struct {
	Name   string  `json:"name"`
	Realm  string  `json:"realm"`
	Class  string  `json:"class"`
	Race   string  `json:"race,omitempty"`
	Level  int     `json:"level,omitempty"`
	Skills []Skill `json:"skills,omitempty"`
	Seen   int64   `json:"seen"`
}

// Outage is a stretch of failed requests, in the companion's clock, UTC.
type Outage struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// SyncRequest is the body of POST /api/companion/sync. Field order is the order the contract's
// test vector uses.
type SyncRequest struct {
	V           int         `json:"v"`
	GameRunning bool        `json:"game_running"`
	Characters  []Character `json:"characters,omitempty"`
	Outage      *Outage     `json:"outage,omitempty"`
	RosterEtag  string      `json:"roster_etag,omitempty"`
}

// VersionOnly is the body of link/status and unlink.
type VersionOnly struct {
	V int `json:"v"`
}

// LinkRequest is the body of the unsigned POST /api/companion/link.
type LinkRequest struct {
	V          int    `json:"v"`
	DeviceName string `json:"device_name"`
	PublicKey  string `json:"public_key"`
}

// LinkResponse answers LinkRequest.
type LinkResponse struct {
	Device    string `json:"device"`
	UserCode  string `json:"user_code"`
	ExpiresIn int    `json:"expires_in"`
	Interval  int    `json:"interval"`
}

// StatusResponse answers link/status.
type StatusResponse struct {
	Linked bool   `json:"linked"`
	Member string `json:"member"`
}

// RosterChar is a character in the member list the hub sends back.
type RosterChar struct {
	Name  string
	Class string
	Level int
	Main  bool
}

// RosterMember is one member of the hub's list.
type RosterMember struct {
	Name       string
	Characters []RosterChar
}

// Roster is the hub's member list.
type Roster struct {
	Members []RosterMember
}

// SyncResponse is the part of the 200 answer of sync the companion uses, with the member list
// already checked by ParseRoster.
type SyncResponse struct {
	RosterEtag string
	Roster     *Roster
}

// ErrRosterShape is returned when the member list is not an object with a members array.
var ErrRosterShape = errors.New("member list has the wrong shape")

// ParseRoster reads the member list the hub sent. It skips an entry outside the contract's
// bounds and keeps the rest, because one long display name on the hub must not turn every sync
// into an outage; it fails only when the shape is wrong.
func ParseRoster(raw []byte) (Roster, error) {
	var top struct {
		Members *[]json.RawMessage `json:"members"`
	}
	if err := json.Unmarshal(raw, &top); err != nil || top.Members == nil {
		return Roster{}, ErrRosterShape
	}
	var r Roster
	for _, m := range *top.Members {
		var in struct {
			Name       string            `json:"name"`
			Characters []json.RawMessage `json:"characters"`
		}
		if json.Unmarshal(m, &in) != nil || in.Name == "" || len(in.Name) > MaxMemberName {
			continue
		}
		mem := RosterMember{Name: in.Name}
		for _, c := range in.Characters {
			if len(mem.Characters) >= MaxMemberChars {
				break
			}
			var ch struct {
				Name  string      `json:"name"`
				Class string      `json:"class"`
				Level json.Number `json:"level"`
				Main  bool        `json:"main"`
			}
			if json.Unmarshal(c, &ch) != nil || ch.Name == "" || len(ch.Name) > MaxFieldBytes || len(ch.Class) > MaxFieldBytes {
				continue
			}
			rc := RosterChar{Name: ch.Name, Class: ch.Class, Main: ch.Main}
			if n, err := ch.Level.Int64(); err == nil && n >= 1 && n <= 999 {
				rc.Level = int(n)
			}
			mem.Characters = append(mem.Characters, rc)
		}
		r.Members = append(r.Members, mem)
	}
	return r, nil
}
