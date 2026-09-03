package store

import (
	"errors"
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	t.Parallel()
	want := TimeIDCursor{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), ID: "019c3d56-7890-7abc-8def-0123456789ab"}
	raw, err := EncodeCursor(want)
	if err != nil {
		t.Fatalf("EncodeCursor: %v", err)
	}
	var got TimeIDCursor
	if err := DecodeCursor(raw, &got); err != nil {
		t.Fatalf("DecodeCursor: %v", err)
	}
	if got != want {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}

func TestDecodeCursorRejectsMalformedValue(t *testing.T) {
	t.Parallel()
	if err := DecodeCursor("not-base64!", &TimeIDCursor{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("DecodeCursor error = %v, want ErrValidation", err)
	}
}
