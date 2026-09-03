package campaign

import (
	"encoding/json"

	"github.com/bubnov-mikhail/ssn688/internal/i18n"
)

// LocText is a scenario localized string object {"en":"...","ru":"..."}.
type LocText i18n.TranslatedText

func (l *LocText) UnmarshalJSON(b []byte) error {
	tt, err := i18n.UnmarshalLocString(b)
	if err != nil {
		return err
	}
	*l = LocText(tt)
	return nil
}

func (l LocText) MarshalJSON() ([]byte, error) {
	return i18n.MarshalLocString(i18n.TranslatedText(l))
}

func (l LocText) GetText(lang string) string {
	return i18n.TranslatedText(l).GetText(lang)
}

func (l LocText) TT() i18n.TranslatedText {
	if l == nil {
		return nil
	}
	return i18n.TranslatedText(l)
}

func locENRU(en, ru string) LocText {
	return LocText(i18n.T(en, ru))
}

// Ensure LocText implements json.Marshaler/Unmarshaler.
var (
	_ json.Marshaler   = LocText{}
	_ json.Unmarshaler = (*LocText)(nil)
)
