package main

import "testing"

func TestUpdateRepositoryURLs(t *testing.T) {
	if updateCheckURL != "https://api.github.com/repos/sqyl2026/hermes-dock-releases/releases/latest" {
		t.Fatalf("updateCheckURL = %q", updateCheckURL)
	}
	if updateRepoURL != "https://github.com/sqyl2026/hermes-dock-releases" {
		t.Fatalf("updateRepoURL = %q", updateRepoURL)
	}
}

func TestIsAllowedUpdateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		allowed bool
	}{
		{
			name:    "release page",
			url:     "https://github.com/sqyl2026/hermes-dock-releases/releases/tag/v1.9.5",
			allowed: true,
		},
		{
			name:    "release asset",
			url:     "https://github.com/sqyl2026/hermes-dock-releases/releases/download/v1.9.5/hermes-dock-v1.9.5-linux-amd64.tar.gz",
			allowed: true,
		},
		{
			name:    "gh proxy asset",
			url:     "https://gh-proxy.com/https://github.com/sqyl2026/hermes-dock-releases/releases/download/v1.9.5/hermes-dock-v1.9.5-linux-amd64.tar.gz",
			allowed: true,
		},
		{
			name:    "ghfast asset",
			url:     "https://ghfast.top/https://github.com/sqyl2026/hermes-dock-releases/releases/download/v1.9.5/hermes-dock-v1.9.5-linux-amd64.tar.gz",
			allowed: true,
		},
		{
			name: "private source release",
			url:  "https://github.com/sqyl2026/hermes-dock/releases/tag/v1.9.5",
		},
		{
			name: "similar release path",
			url:  "https://github.com/sqyl2026/hermes-dock-releases/releases-archive/tag/v1.9.5",
		},
		{
			name: "other repository",
			url:  "https://github.com/sqyl2026/other/releases/tag/v1.9.5",
		},
		{
			name: "insecure scheme",
			url:  "http://github.com/sqyl2026/hermes-dock-releases/releases/tag/v1.9.5",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isAllowedUpdateURL(test.url); got != test.allowed {
				t.Fatalf("isAllowedUpdateURL(%q) = %v, want %v", test.url, got, test.allowed)
			}
		})
	}
}
