package model

import (
	"net/url"
	"testing"
)

func TestParsePaginationDefaults(t *testing.T) {
	got, err := ParsePagination(url.Values{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Limit != DefaultLimit || got.Offset != 0 {
		t.Fatalf("unexpected pagination: %+v", got)
	}
}

func TestParsePaginationRejectsBadLimit(t *testing.T) {
	_, err := ParsePagination(url.Values{"limit": []string{"101"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
