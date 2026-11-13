package savedvars

import (
	"fmt"
	"io"
	"io/fs"
	"sort"

	"github.com/Atomtomate/wow-forever-addons/companion/internal/wire"
)

// FormatVersion is the version of the §1 file this reader knows; the sync body's version is
// another number (wire.Version) and the two change separately.
const FormatVersion = 1

// FilePattern finds one file per WoW account, relative to the game folder.
const FilePattern = "WTF/Account/*/SavedVariables/ForeverFriends.lua"

// File is one account's file as read.
type File struct {
	Path string
	// Fingerprint changes when the file is rewritten; it is compared, never interpreted.
	Fingerprint string
	// OK is false when the file could not be read or parsed this time (the client may be in the
	// middle of writing it); such a file contributes nothing and is read again at the next tick.
	OK      bool
	Problem string
}

// Snapshot is what the files held at one moment.
type Snapshot struct {
	Files []File
	// Characters is the merge over every OK file, as the sync sends it: newest `seen` first, at
	// most wire.MaxCharacters, skill names dropped.
	Characters []wire.Character
}

// Read reads every account's file under fsys, which is rooted at the game folder.
func Read(fsys fs.FS) Snapshot {
	var snap Snapshot
	paths, _ := fs.Glob(fsys, FilePattern)
	sort.Strings(paths)
	merged := map[string]wire.Character{}
	for _, p := range paths {
		f := File{Path: p}
		chars, fp, err := readOne(fsys, p)
		f.Fingerprint = fp
		if err != nil {
			f.Problem = err.Error()
		} else {
			f.OK = true
			for k, c := range chars {
				if old, ok := merged[k]; !ok || c.Seen > old.Seen {
					merged[k] = c
				}
			}
		}
		snap.Files = append(snap.Files, f)
	}
	for _, c := range merged {
		snap.Characters = append(snap.Characters, c)
	}
	sort.Slice(snap.Characters, func(i, j int) bool {
		a, b := snap.Characters[i], snap.Characters[j]
		if a.Seen != b.Seen {
			return a.Seen > b.Seen
		}
		return a.Name+"\x00"+a.Realm < b.Name+"\x00"+b.Realm
	})
	if len(snap.Characters) > wire.MaxCharacters {
		snap.Characters = snap.Characters[:wire.MaxCharacters]
	}
	return snap
}

func readOne(fsys fs.FS, path string) (map[string]wire.Character, string, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	fp := fmt.Sprintf("%d-%d", st.Size(), st.ModTime().UnixNano())
	if st.Size() > MaxFileBytes {
		return nil, fp, fmt.Errorf("file is larger than %d bytes", MaxFileBytes)
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil {
		return nil, fp, err
	}
	top, err := Parse(b)
	if err != nil {
		return nil, fp, err
	}
	chars, err := characters(top)
	return chars, fp, err
}

func asInt(v Value) (int, bool) {
	f, ok := v.(float64)
	if !ok || f != float64(int(f)) || f < 0 || f > 1<<31 {
		return 0, false
	}
	return int(f), true
}

// characters pulls the entries of ForeverFriendsDB. Entries that lack name, realm or class, or
// whose name or realm is over the limit, are left out; skills keep only id, rank and max.
func characters(top map[string]Value) (map[string]wire.Character, error) {
	db, ok := top["ForeverFriendsDB"].(*Table)
	if !ok {
		return nil, fmt.Errorf("no ForeverFriendsDB table")
	}
	if v, ok := asInt(db.Get("version")); !ok || v != FormatVersion {
		return nil, fmt.Errorf("unknown format version, this companion reads %d", FormatVersion)
	}
	out := map[string]wire.Character{}
	chars, ok := db.Get("characters").(*Table)
	if !ok {
		return out, nil
	}
	for i, k := range chars.Keys {
		key, ok := k.(string)
		t, ok2 := chars.Vals[i].(*Table)
		if !ok || !ok2 {
			continue
		}
		name, _ := t.Get("name").(string)
		realm, _ := t.Get("realm").(string)
		class, _ := t.Get("class").(string)
		if name == "" || realm == "" || class == "" || len(name) > wire.MaxFieldBytes || len(realm) > wire.MaxFieldBytes || len(class) > wire.MaxFieldBytes {
			continue
		}
		c := wire.Character{Name: name, Realm: realm, Class: class}
		if r, ok := t.Get("race").(string); ok && len(r) <= wire.MaxFieldBytes {
			c.Race = r
		}
		if n, ok := asInt(t.Get("level")); ok {
			c.Level = n
		}
		if n, ok := asInt(t.Get("seen")); ok {
			c.Seen = int64(n)
		}
		if sk, ok := t.Get("skills").(*Table); ok {
			for _, s := range sk.List() {
				st, ok := s.(*Table)
				if !ok || len(c.Skills) >= wire.MaxSkills {
					continue
				}
				id, ok := asInt(st.Get("id"))
				if !ok {
					continue
				}
				rank, _ := asInt(st.Get("rank"))
				max, _ := asInt(st.Get("max"))
				c.Skills = append(c.Skills, wire.Skill{ID: id, Rank: rank, Max: max})
			}
		}
		out[key] = c
	}
	return out, nil
}
