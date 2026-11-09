package main

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"

	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// writeSafe replaces a file so that a phone that dies at any moment leaves
// the old content or the new one. Without the Sync, the rename can reach the
// disk before the data does, and what comes back is an empty file.
func writeSafe(path string, data []byte) error {
	return replaceSafe(path, func(tmp *os.File) error {
		_, err := tmp.Write(data)
		return err
	})
}

// replaceSafe writes a file through a temporary one next to it, made by fill,
// and puts it in place only once it is whole on the disk. The temporary file
// has a name of its own for each call, so two saves of the same note at once
// never write into each other, and it starts with a dot: the sync and the
// lists of notes leave it out (notesync ignores hidden names), so one seen in
// the middle of a save, or left behind by an app that died, never travels.
func replaceSafe(path string, fill func(tmp *os.File) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	err = fill(tmp)
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(name, path)
	}
	if err != nil {
		os.Remove(name)
		return err
	}
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		dir.Sync()
		dir.Close()
	}
	return nil
}

// setPropsSafe changes properties of the note at path, as
// obsidian.SetProperties does, but on a copy that takes the note's place only
// when whole: SetProperties writes the file in place, and a phone that died
// in the middle would leave half a note.
func setPropsSafe(path string, props ...obsidian.Prop) error {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	propsErr, err := saveWithProps(path, data, func(string) []obsidian.Prop { return props })
	return cmp.Or(err, propsErr)
}

// saveWithProps writes body to path with the properties props gives (it
// reads the note being written, by its temporary path), all at once. When the
// note's properties can't be read (a block broken by hand), the text is still
// kept as it came, without the new properties, and propsErr says why.
func saveWithProps(path string, body []byte, props func(tmp string) []obsidian.Prop) (propsErr, err error) {
	err = replaceSafe(path, func(tmp *os.File) error {
		if _, err := tmp.Write(body); err != nil {
			return err
		}
		// SetProperties opens the file by its name and writes it again: the
		// temporary file is synced and renamed after it.
		if propsErr = obsidian.SetProperties(tmp.Name(), props(tmp.Name())...); propsErr == nil {
			return nil
		}
		if err := tmp.Truncate(0); err != nil {
			return err
		}
		_, err := tmp.WriteAt(body, 0)
		return err
	})
	return propsErr, err
}
