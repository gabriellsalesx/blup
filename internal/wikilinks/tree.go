package wikilinks

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Report summarises a ConvertTree run.
type Report struct {
	Scanned  int      // Markdown files read
	Files    int      // files changed (or that would change, in a dry run)
	Changed  []string // vault-relative paths of those files
	Changes  []Change
	Problems []Problem
}

// ConvertTree converts the wikilinks of every .md file under subdir (a
// vault-relative folder; "" means the whole vault), skipping dot
// directories. Links to files outside subdir are converted but reported as
// warnings. With dryRun nothing is written; otherwise each changed file is
// replaced atomically (temp file + rename) keeping its mode. Line endings are
// preserved because only the wikilinks themselves are rewritten.
func ConvertTree(idx *Index, vaultRoot, subdir string, dryRun bool) (Report, error) {
	var rep Report
	files, err := markdownFiles(vaultRoot, subdir)
	if err != nil {
		return rep, err
	}
	scope := ""
	if subdir != "" {
		scope = path.Clean(filepath.ToSlash(subdir))
	}
	for _, rel := range files {
		full := filepath.Join(vaultRoot, filepath.FromSlash(rel))
		data, err := os.ReadFile(full)
		if err != nil {
			return rep, err
		}
		rep.Scanned++
		out, ch, pr := convertText(idx, rel, string(data), scope)
		rep.Changes = append(rep.Changes, ch...)
		rep.Problems = append(rep.Problems, pr...)
		if out == string(data) {
			continue
		}
		rep.Files++
		rep.Changed = append(rep.Changed, rel)
		if !dryRun {
			if err := writeAtomic(full, []byte(out)); err != nil {
				return rep, fmt.Errorf("writing %s: %w", rel, err)
			}
		}
	}
	return rep, nil
}

// Backup writes destZip with every .md file under subdir ("" = whole vault),
// stored with its vault-relative path. Call it before ConvertTree writes.
// To back up only the files a conversion will change, run ConvertTree with
// dryRun and pass Report.Changed to BackupFiles.
func Backup(vaultRoot, subdir, destZip string) error {
	files, err := markdownFiles(vaultRoot, subdir)
	if err != nil {
		return err
	}
	return BackupFiles(vaultRoot, files, destZip)
}

// BackupFiles writes destZip with the given vault-relative files. The zip is
// written to a temporary file first and renamed into place.
func BackupFiles(vaultRoot string, files []string, destZip string) (err error) {
	dir := filepath.Dir(destZip)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".blup-backup-*.zip")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	zw := zip.NewWriter(tmp)
	for _, rel := range files {
		if err = addToZip(zw, vaultRoot, rel); err != nil {
			return err
		}
	}
	if err = zw.Close(); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), destZip)
}

func addToZip(zw *zip.Writer, root, rel string) error {
	full := filepath.Join(root, filepath.FromSlash(rel))
	f, err := os.Open(full)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	hdr.Name = filepath.ToSlash(rel)
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

// markdownFiles lists the vault-relative paths of the .md files under
// subdir, skipping dot directories.
func markdownFiles(vaultRoot, subdir string) ([]string, error) {
	sub := path.Clean("/" + filepath.ToSlash(subdir))[1:]
	if subdir != "" && (sub == "" || strings.HasPrefix(filepath.ToSlash(filepath.Clean(subdir)), "..")) {
		return nil, fmt.Errorf("invalid subdir %q", subdir)
	}
	base := filepath.Join(vaultRoot, filepath.FromSlash(sub))
	fi, err := os.Stat(base)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", base)
	}
	var out []string
	err = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != base && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") || !isNote(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(vaultRoot, p)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}

// writeAtomic replaces file with data through a temporary file in the same
// directory, keeping the original permissions.
func writeAtomic(file string, data []byte) (err error) {
	fi, err := os.Stat(file)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(file), ".blup-*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Chmod(fi.Mode().Perm()); err != nil && !errors.Is(err, errors.ErrUnsupported) {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), file)
}
