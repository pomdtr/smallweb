package cmd

import (
	"testing"
)

func TestLookupApp(t *testing.T) {
	type TestCaseWant struct {
		app      string
		redirect bool
		ok       bool
	}
	type TestCase struct {
		name          string
		host          string
		domain        string
		customDomains map[string]string
		want          TestCaseWant
	}

	cases := []TestCase{
		{
			name:   "apex domain",
			host:   "smallweb.run",
			domain: "smallweb.run",
			customDomains: map[string]string{
				"smallweb.cloud":      "cloud",
				"custom.smallweb.run": "www",
				"pomdtr.me":           "*",
			},
			want: TestCaseWant{
				app:      "www",
				redirect: true,
				ok:       true,
			},
		},
		{
			name:   "subdomain",
			host:   "example.smallweb.run",
			domain: "smallweb.run",
			customDomains: map[string]string{
				"smallweb.cloud":      "cloud",
				"custom.smallweb.run": "www",
				"pomdtr.me":           "*",
			},
			want: TestCaseWant{
				app:      "example",
				redirect: false,
				ok:       true,
			},
		},
		{
			name:   "custom subdomain",
			host:   "custom.smallweb.run",
			domain: "smallweb.run",
			customDomains: map[string]string{
				"smallweb.cloud":      "cloud",
				"custom.smallweb.run": "www",
				"pomdtr.me":           "*",
			},
			want: TestCaseWant{
				app:      "www",
				redirect: false,
				ok:       true,
			},
		},
		{
			name:   "custom domain exact match",
			host:   "smallweb.cloud",
			domain: "smallweb.run",
			customDomains: map[string]string{
				"smallweb.cloud":      "cloud",
				"custom.smallweb.run": "www",
				"pomdtr.me":           "*",
			},
			want: TestCaseWant{
				app:      "cloud",
				redirect: false,
				ok:       true,
			},
		},
		{
			name:   "custom domain wildcard apex domain",
			host:   "pomdtr.me",
			domain: "smallweb.run",
			customDomains: map[string]string{
				"smallweb.cloud":      "cloud",
				"custom.smallweb.run": "www",
				"pomdtr.me":           "*",
			},
			want: TestCaseWant{
				app:      "www",
				redirect: true,
				ok:       true,
			},
		},
		{
			name:   "custom domain wildcard subdomain",
			host:   "example.pomdtr.me",
			domain: "smallweb.run",
			customDomains: map[string]string{
				"smallweb.cloud":      "cloud",
				"custom.smallweb.run": "www",
				"pomdtr.me":           "*",
			},
			want: TestCaseWant{
				app:      "example",
				redirect: false,
				ok:       true,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app, redirect, ok := lookupApp(c.host, c.domain, c.customDomains)
			if app != c.want.app || redirect != c.want.redirect || ok != c.want.ok {
				t.Errorf("lookupApp() = %v, %v, %v; want %v, %v, %v", app, redirect, ok, c.want.app, c.want.redirect, c.want.ok)
			}
		})
	}
}
