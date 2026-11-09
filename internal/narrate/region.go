package narrate

import (
	"regexp"
	"strings"
)

// A voice's region, for a list where two voices share a name in the same
// language ("pt-BR-Grant" and "pt-PT-Grant" both read Portuguese): the
// list then says "Grant (Brasil)" and "Grant (Portugal)". Neither is
// hidden.

var reRegion = regexp.MustCompile(`^[a-z]{2}[-_]([A-Z]{2})[-_]`)

// VoiceRegion is a voice's region code ("BR" for "pt-BR-Grant:MAI-Voice-2.1"),
// "" when its name carries none.
func VoiceRegion(voice string) string {
	if m := reRegion.FindStringSubmatch(voice); m != nil {
		return m[1]
	}
	return ""
}

// regionNames are the regions' names in each language of the Studio.
var regionNames = map[string]map[string]string{
	"pt": {"BR": "Brasil", "PT": "Portugal", "US": "EUA", "GB": "Reino Unido", "MX": "México", "ES": "Espanha"},
	"en": {"BR": "Brazil", "PT": "Portugal", "US": "US", "GB": "UK", "MX": "Mexico", "ES": "Spain"},
	"es": {"BR": "Brasil", "PT": "Portugal", "US": "EE. UU.", "GB": "Reino Unido", "MX": "México", "ES": "España"},
}

// RegionName is a region's name in lang (the Studio's language); the code
// itself when the region is not known.
func RegionName(code, lang string) string {
	if n := regionNames[lang][code]; n != "" {
		return n
	}
	if n := regionNames["en"][code]; n != "" && regionNames[lang] == nil {
		return n
	}
	return code
}

// VoiceLabels names each voice of a list: its name, and its region when
// another voice of the list has the same name (lang: the Studio's
// language). A voice without a region keeps its name alone.
func VoiceLabels(voices []string, lang string) []string {
	count := map[string]int{}
	for _, v := range voices {
		count[strings.ToLower(VoiceName(v))]++
	}
	out := make([]string, len(voices))
	for i, v := range voices {
		name := VoiceName(v)
		if r := VoiceRegion(v); r != "" && count[strings.ToLower(name)] > 1 {
			name += " (" + RegionName(r, lang) + ")"
		}
		out[i] = name
	}
	return out
}
