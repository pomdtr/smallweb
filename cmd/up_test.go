package cmd_test

import (
	"testing"

	"github.com/pomdtr/smallweb/cmd"
)

func TestSplitHost(t *testing.T) {
	var tests = []struct {
		host      string
		domain    string
		subdomain string
	}{
		{"example.com", "example.com", ""},
		{"sub.example.com", "example.com", "sub"},
		{"test.localhost", "localhost", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			domain, subdomain, err := cmd.SplitHost(tt.host)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if domain != tt.domain {
				t.Errorf("expected domain %q, got %q", tt.domain, domain)
			}

			if subdomain != tt.subdomain {
				t.Errorf("expected subdomain %q, got %q", tt.subdomain, subdomain)
			}
		})
	}

}
