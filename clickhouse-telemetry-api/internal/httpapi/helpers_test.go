package httpapi

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseWindowDefaultsToPreviousHour(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	request := httptest.NewRequest("GET", "/", nil)
	from, to, err := parseWindow(request, now)
	if err != nil {
		t.Fatalf("parseWindow: %v", err)
	}
	if !to.Equal(now) || !from.Equal(now.Add(-time.Hour)) {
		t.Fatalf("window = %s to %s", from, to)
	}
}

func TestParseWindowRejectsLargeRange(t *testing.T) {
	t.Parallel()
	request := httptest.NewRequest("GET", "/?from=2026-01-01T00:00:00Z&to=2026-03-01T00:00:00Z", nil)
	if _, _, err := parseWindow(request, time.Now()); err == nil {
		t.Fatal("parseWindow accepted a range longer than 31 days")
	}
}

func TestCursorRoundTrip(t *testing.T) {
	t.Parallel()
	timestamp := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	items := []struct {
		ID   string
		Time time.Time
	}{
		{ID: "019c3d56-7890-7abc-8def-0123456789ab", Time: timestamp},
		{ID: "019c3d56-7890-7abc-8def-0123456789ac", Time: timestamp.Add(-time.Second)},
	}
	result, err := buildPage(items, 1, func(item struct {
		ID   string
		Time time.Time
	}) listCursor {
		return listCursor{Time: item.Time, ID: item.ID}
	})
	if err != nil {
		t.Fatalf("buildPage: %v", err)
	}
	if result.NextCursor == nil || len(result.Items) != 1 {
		t.Fatalf("page = %#v", result)
	}
	request := httptest.NewRequest("GET", "/?cursor="+*result.NextCursor, nil)
	limit, cursor, err := parseList(request)
	if err != nil {
		t.Fatalf("parseList: %v", err)
	}
	if limit != defaultPageLimit || cursor == nil || cursor.ID != items[0].ID || !cursor.Time.Equal(timestamp) {
		t.Fatalf("cursor = %#v, limit = %d", cursor, limit)
	}
}

func TestParseListRejectsMalformedCursor(t *testing.T) {
	t.Parallel()
	request := httptest.NewRequest("GET", "/?cursor=not-base64!", nil)
	if _, _, err := parseList(request); err == nil {
		t.Fatal("parseList accepted malformed cursor")
	}
}
