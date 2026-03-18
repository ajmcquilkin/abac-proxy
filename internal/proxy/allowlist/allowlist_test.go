package allowlist

import (
	"testing"

	"github.com/abac/proxy/internal/api"
)

func newFromEntries(entries []api.HostEntry) Allowlist {
	return &allowlist{AllowedHosts: entries}
}

func TestFindHost(t *testing.T) {
	tests := []struct {
		name       string
		entries    []api.HostEntry
		host       string
		wantScheme string
		wantFound  bool
	}{
		{
			"exact match",
			[]api.HostEntry{{Host: "api.example.com", Scheme: "https"}},
			"api.example.com",
			"https",
			true,
		},
		{
			"exact match case insensitive",
			[]api.HostEntry{{Host: "API.Example.com", Scheme: "https"}},
			"api.example.com",
			"https",
			true,
		},
		{
			"wildcard match",
			[]api.HostEntry{{Host: "*.example.com", Scheme: "https"}},
			"api.example.com",
			"https",
			true,
		},
		{
			"wildcard bare domain",
			[]api.HostEntry{{Host: "*.example.com", Scheme: "https"}},
			"example.com",
			"https",
			true,
		},
		{
			"no match",
			[]api.HostEntry{{Host: "api.example.com", Scheme: "https"}},
			"other.com",
			"",
			false,
		},
		{
			"http scheme",
			[]api.HostEntry{{Host: "localhost", Scheme: "http"}},
			"localhost",
			"http",
			true,
		},
		{
			"trims whitespace",
			[]api.HostEntry{{Host: "api.example.com", Scheme: "https"}},
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
	c := newFromEntries([]api.HostEntry{
		{Host: "api.example.com", Scheme: "https"},
	})

	if !c.IsAllowed("api.example.com") {
		t.Error("expected api.example.com to be allowed")
	}
	if c.IsAllowed("other.com") {
		t.Error("expected other.com to not be allowed")
	}
}

func TestFromEntries(t *testing.T) {
	t.Run("valid entries", func(t *testing.T) {
		al, err := FromEntries([]api.HostEntry{
			{Host: "api.example.com", Scheme: "https"},
			{Host: "localhost", Scheme: "http"},
		})
		if err != nil {
			t.Fatalf("FromEntries() error = %v", err)
		}
		if !al.IsAllowed("api.example.com") {
			t.Error("expected api.example.com to be allowed")
		}
		if !al.IsAllowed("localhost") {
			t.Error("expected localhost to be allowed")
		}
	})

	t.Run("empty entries", func(t *testing.T) {
		_, err := FromEntries([]api.HostEntry{})
		if err == nil {
			t.Fatal("expected error for empty entries")
		}
	})

	t.Run("default scheme", func(t *testing.T) {
		al, err := FromEntries([]api.HostEntry{{Host: "api.example.com"}})
		if err != nil {
			t.Fatalf("FromEntries() error = %v", err)
		}
		scheme, found := al.FindHost("api.example.com")
		if !found {
			t.Fatal("expected host to be found")
		}
		if scheme != "https" {
			t.Errorf("scheme = %q, want %q", scheme, "https")
		}
	})
}

func TestGetHostList(t *testing.T) {
	c := newFromEntries([]api.HostEntry{
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
