package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// LocString is a scenario JSON localized string: {"en":"...","ru":"..."}.
// Alias of TranslatedText with strict object unmarshaling (format major 3).
type LocString = TranslatedText

// UnmarshalLocString parses a JSON object of language→text into TranslatedText.
func UnmarshalLocString(data []byte) (TranslatedText, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	if data[0] == '"' {
		return nil, fmt.Errorf("localized string must be an object {\"en\":\"...\",\"ru\":\"...\"}, not a plain string")
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return TranslatedText{}, nil
	}
	return TranslatedText(m), nil
}

// MarshalLocString encodes TranslatedText as a JSON object.
func MarshalLocString(t TranslatedText) ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]string(t))
}
