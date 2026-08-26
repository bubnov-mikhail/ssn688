// Package i18n provides bilingual (EN/RU) text and audio path helpers.
package i18n

import "sync/atomic"

const (
	LangEN = "en"
	LangRU = "ru"
)

// SupportedLangs is the ordered list of UI/scenario languages (extend later).
var SupportedLangs = []string{LangEN, LangRU}

// NativeName is the language label shown on its own language (settings buttons).
func NativeName(lang string) string {
	switch lang {
	case LangRU:
		return "РУССКИЙ"
	case LangEN:
		return "ENGLISH"
	default:
		return lang
	}
}

// NormalizeLang returns a supported code or LangEN.
func NormalizeLang(lang string) string {
	for _, l := range SupportedLangs {
		if lang == l {
			return lang
		}
	}
	return LangEN
}

var currentLang atomic.Value // string

func init() {
	currentLang.Store(LangEN)
}

// SetLang updates the process-wide UI language (from settings).
func SetLang(lang string) {
	currentLang.Store(NormalizeLang(lang))
}

// CurrentLang returns the process-wide UI language.
func CurrentLang() string {
	if v := currentLang.Load(); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return LangEN
}

// TranslatedText maps language code → localized string.
type TranslatedText map[string]string

// T builds a bilingual entry (English + Russian).
func T(en, ru string) TranslatedText {
	return TranslatedText{LangEN: en, LangRU: ru}
}

// GetText returns the string for lang, falling back to en, then any non-empty value.
func (t TranslatedText) GetText(lang string) string {
	if t == nil {
		return ""
	}
	lang = NormalizeLang(lang)
	if s := t[lang]; s != "" {
		return s
	}
	if s := t[LangEN]; s != "" {
		return s
	}
	for _, code := range SupportedLangs {
		if s := t[code]; s != "" {
			return s
		}
	}
	for _, s := range t {
		if s != "" {
			return s
		}
	}
	return ""
}

// TranslatedAudio maps language code → voice clip path (e.g. "capt/comm_message").
// Russian assets live under the "ru/" prefix (e.g. "ru/capt/comm_message").
type TranslatedAudio map[string]string

// A builds EN + RU audio paths. Missing RU WAVs fall back to EN at PlayClip time.
func A(enPath string) TranslatedAudio {
	return TranslatedAudio{
		LangEN: enPath,
		LangRU: "ru/" + enPath,
	}
}

// GetWav returns the clip path for lang, falling back to en, then any non-empty path.
func (t TranslatedAudio) GetWav(lang string) string {
	if t == nil {
		return ""
	}
	lang = NormalizeLang(lang)
	if s := t[lang]; s != "" {
		return s
	}
	if s := t[LangEN]; s != "" {
		return s
	}
	for _, code := range SupportedLangs {
		if s := t[code]; s != "" {
			return s
		}
	}
	for _, s := range t {
		if s != "" {
			return s
		}
	}
	return ""
}
