package wikilinks

// Shorthands the tests use.

// ConvertText is convertText for the whole vault.
func ConvertText(idx *Index, notePath, text string) (out string, changes []Change, problems []Problem) {
	return convertText(idx, notePath, text, "")
}

// Files returns the vault-relative, slash-separated paths of all indexed files.
func (idx *Index) Files() []string { return append([]string(nil), idx.files...) }

// Backup writes destZip with every .md file under subdir ("" = whole vault).
func Backup(vaultRoot, subdir, destZip string) error {
	files, err := markdownFiles(vaultRoot, subdir)
	if err != nil {
		return err
	}
	return BackupFiles(vaultRoot, files, destZip)
}
