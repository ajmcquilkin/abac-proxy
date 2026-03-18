package allowlist

import "testing"

func newFromEntries(entries []HostEntry) Allowlist {
	return &allowlist{AllowedHosts: entries}
}

func TestFindHost(t *testing.T) {
	tests := []struct {
		name       string
		entries    []HostEntry
		host       string
		wantScheme string
		wantFound  bool
	}{
		{
			"exact match",
			[]HostEntry{{Host: "api.example.com", Scheme: "https"}},
			"api.example.com",
			"https",
			true,
		},
		{
			"exact match case insensitive",
			[]HostEntry{{Host: "API.Example.com", Scheme: "https"}},
			"api.example.com",
			"https",
			true,
		},
		{
			"wildcard match",
			[]HostEntry{{Host: "*.example.com", Scheme: "https"}},
			"api.example.com",
			"https",
			true,
		},
		{
			"wildcard bare domain",
			[]HostEntry{{Host: "*.example.com", Scheme: "https"}},
			"example.com",
			"https",
			true,
		},
		{
			"no match",
			[]HostEntry{{Host: "api.example.com", Scheme: "https"}},
			"other.com",
			"",
			false,
		},
		{
			"http scheme",
			[]HostEntry{{Host: "localhost", Scheme: "http"}},
			"localhost",
			"http",
			true,
		},
		{
			"trims whitespace",
			[]HostEntry{{Host: "api.example.com", Scheme: "https"}},
			"  api.example.com  ",
			"https",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newFromEntries(tt.entries)
			scheme, found := c.FindHost(tt.host)
			if found != tt.wantFound {
				t.Fatalf("FindHost(%q) found = %v, want %v", tt.host, found, tt.wantFound)
			}
			if scheme != tt.wantScheme {
				t.Errorf("FindHost(%q) scheme = %q, want %q", tt.host, scheme, tt.wantScheme)
			}
		})
	}
}

func TestIsAllowed(t *testing.T) {
	c := newFromEntries([]HostEntry{
		{Host: "api.example.com", Scheme: "https"},
	})

	if !c.IsAllowed("api.example.com") {
		t.Error("expected api.example.com to be allowed")
	}
	if c.IsAllowed("other.com") {
		t.Error("expected other.com to not be allowed")
	}
}

func TestGetHostList(t *testing.T) {
	c := newFromEntries([]HostEntry{
		{Host: "api.example.com", Scheme: "https"},
		{Host: "localhost", Scheme: "http"},
	})

	got := c.GetHostList()
	want := []string{"https://api.example.com", "http://localhost"}
	if len(got) != len(want) {
		t.Fatalf("GetHostList() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("GetHostList()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
