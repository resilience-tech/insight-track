package store

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func EncodeCursor(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func DecodeCursor(raw string, target any) error {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return ErrValidation
	}
	if err := json.Unmarshal(decoded, target); err != nil {
		return ErrValidation
	}
	return nil
}
