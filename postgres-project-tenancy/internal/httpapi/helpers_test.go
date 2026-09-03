package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestParseIfMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		header  string
		want    int64
		wantErr bool
	}{
		{name: "valid", header: `"17"`, want: 17},
		{name: "missing", wantErr: true},
		{name: "weak", header: `W/"17"`, wantErr: true},
		{name: "wildcard", header: `*`, wantErr: true},
		{name: "zero", header: `"0"`, wantErr: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest("PATCH", "/", nil)
			request.Header.Set("If-Match", test.header)
			got, err := parseIfMatch(request)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseIfMatch error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("parseIfMatch = %d, want %d", got, test.want)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	t.Parallel()
	if !validateUUID("019c3d56-7890-7abc-8def-0123456789ab") {
		t.Fatal("valid UUID was rejected")
	}
	if validateUUID("../../etc/passwd") {
		t.Fatal("invalid UUID was accepted")
	}
}

func TestValidateJSONObject(t *testing.T) {
	t.Parallel()
	if !validateJSONObject([]byte(`{"name":"value"}`)) {
		t.Fatal("object was rejected")
	}
	for _, raw := range [][]byte{[]byte(`null`), []byte(`[]`), []byte(`"value"`)} {
		if validateJSONObject(raw) {
			t.Fatalf("non-object %s was accepted", raw)
		}
	}
}
