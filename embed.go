// Package foreverfriends carries the ForeverFriends addon inside the companion executable, so
// that nothing has to be downloaded or unpacked (docs/DECISIONS.md D7). The globs name the
// files an addon folder can hold that the game loads; its README and `spec/` are not shipped.
package foreverfriends

import (
	"embed"
	"io/fs"
)

//go:embed addons/ForeverFriends/*.toc addons/ForeverFriends/*.lua
var files embed.FS

// Addon returns the addon's files, rooted at the folder that holds ForeverFriends.toc.
func Addon() fs.FS {
	sub, err := fs.Sub(files, "addons/ForeverFriends")
	if err != nil {
		panic(err) // the path is a constant of this file
	}
	return sub
}
