package install_test

import (
	"io/fs"
	"regexp"
	"testing"

	foreverfriends "github.com/Atomtomate/wow-forever-addons"
	"github.com/Atomtomate/wow-forever-addons/companion/internal/install"
)

// The addon the executable carries must hold every file its own .toc lists, or setup would
// install an addon the game cannot load.
func TestEmbeddedAddonHoldsEveryFileItsTocLists(t *testing.T) {
	src := foreverfriends.Addon()
	toc, err := install.ReadToc(src)
	if err != nil {
		t.Fatal(err)
	}
	// The data addon's .toc copies this value without checking it.
	if !regexp.MustCompile(`^[0-9]+(,[0-9]+)*$`).MatchString(toc.Interface) {
		t.Fatalf("the embedded Interface value %q is not a number list", toc.Interface)
	}
	if len(toc.Files) == 0 {
		t.Fatal("the embedded .toc lists no files")
	}
	for _, f := range toc.Files {
		if _, err := fs.Stat(src, f); err != nil {
			t.Errorf("%s is listed in the .toc and not embedded: %v", f, err)
		}
	}
}
