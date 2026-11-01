//go:build !windows && !darwin

package i18n

func systemLocales() []string { return envLocales() }
