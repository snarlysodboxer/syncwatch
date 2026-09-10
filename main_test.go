package main

import "testing"

func TestNormalizeArgoCDURL(t *testing.T) {
	for _, tc := range []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "unset", raw: "", want: ""},
		{name: "https", raw: "https://argocd.example.com", want: "https://argocd.example.com"},
		{name: "http", raw: "http://argocd.example.com", want: "http://argocd.example.com"},
		{name: "trailing slash is stripped", raw: "https://argocd.example.com/", want: "https://argocd.example.com"},
		{name: "multiple trailing slashes are stripped", raw: "https://argocd.example.com///", want: "https://argocd.example.com"},
		{name: "subpath is kept", raw: "https://example.com/argocd/", want: "https://example.com/argocd"},
		{name: "port is kept", raw: "http://localhost:8080", want: "http://localhost:8080"},
		{name: "javascript scheme is rejected", raw: "javascript:alert(1)", wantErr: true},
		{name: "unknown scheme is rejected", raw: "ftp://argocd.example.com", wantErr: true},
		{name: "missing scheme is rejected", raw: "argocd.example.com", wantErr: true},
		{name: "missing host is rejected", raw: "https://", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeArgoCDURL(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("normalizeArgoCDURL(%q) = %q, want an error", tc.raw, got)
				}

				return
			}
			if err != nil {
				t.Fatalf("normalizeArgoCDURL(%q) returned error: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Errorf("normalizeArgoCDURL(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
