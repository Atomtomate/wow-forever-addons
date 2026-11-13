// Package atomicfile writes a file so that a reader never sees half of it: beside the file,
// then renamed over it. The modes are the caller's, because the config folder is private and
// the addon folders the game reads are not.
package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
)

// Write creates the parent folders with dirMode and replaces path with data.
func Write(path string, data []byte, dirMode, fileMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	serr := f.Sync()
	cerr := f.Close()
	if err := errors.Join(werr, serr, cerr); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
