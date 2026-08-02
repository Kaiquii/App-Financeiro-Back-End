package pagination

import (
	"net/url"
	"testing"
)

func TestParseKeepsLegacyModeWithoutParameters(t *testing.T) {
	params, err := Parse(url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	if params.Enabled {
		t.Fatal("pagination should remain disabled without page or limit")
	}
}

func TestParsePaginationParameters(t *testing.T) {
	tests := []struct {
		name   string
		values url.Values
		page   int
		limit  int
		offset int
	}{
		{name: "page and limit", values: url.Values{"page": {"3"}, "limit": {"25"}}, page: 3, limit: 25, offset: 50},
		{name: "page only", values: url.Values{"page": {"2"}}, page: 2, limit: DefaultLimit, offset: 20},
		{name: "limit only", values: url.Values{"limit": {"50"}}, page: 1, limit: 50, offset: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params, err := Parse(test.values)
			if err != nil {
				t.Fatal(err)
			}
			if !params.Enabled || params.Page != test.page || params.Limit != test.limit || params.Offset != test.offset {
				t.Fatalf("unexpected params: %+v", params)
			}
		})
	}
}

func TestParseRejectsInvalidParameters(t *testing.T) {
	tests := []url.Values{
		{"page": {""}},
		{"page": {"0"}},
		{"page": {"abc"}},
		{"limit": {"0"}},
		{"limit": {"101"}},
		{"limit": {"abc"}},
	}
	for _, values := range tests {
		if _, err := Parse(values); err == nil {
			t.Fatalf("expected validation error for %v", values)
		}
	}
}

func TestNewMetadata(t *testing.T) {
	metadata := NewMetadata(Params{Enabled: true, Page: 2, Limit: 20}, 45)
	if metadata.TotalPages != 3 || !metadata.HasNext || !metadata.HasPrevious {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}

	empty := NewMetadata(Params{Enabled: true, Page: 1, Limit: 20}, 0)
	if empty.TotalPages != 0 || empty.HasNext || empty.HasPrevious {
		t.Fatalf("unexpected empty metadata: %+v", empty)
	}
}
