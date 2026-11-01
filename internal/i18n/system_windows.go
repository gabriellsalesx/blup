//go:build windows

package i18n

import "golang.org/x/sys/windows"

// The user's preferred UI languages ("pt-BR", "en-US"...).
func systemLocales() []string {
	out := envLocales()
	if langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME); err == nil {
		out = append(langs, out...)
	}
	return out
}
