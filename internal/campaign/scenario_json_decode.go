package campaign

import (
	"bytes"
	"encoding/json"
)

func decodeJSONStrict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
