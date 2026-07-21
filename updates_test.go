package main

import (
	"reflect"
	"testing"
)

func TestUpdateRepositoryURLs(t *testing.T) {
	if updateCheckURL != "https://api.github.com/repos/sqyl2026/hermes-dock-releases/releases/latest" {
		t.Fatalf("updateCheckURL = %q", updateCheckURL)
	}
	if updateRepoURL != "https://github.com/sqyl2026/hermes-dock-releases" {
		t.Fatalf("updateRepoURL = %q", updateRepoURL)
	}
}

func TestUpdateSourcePriority(t *testing.T) {
	wantChecks := []string{
		"https://gh-proxy.com/" + updateCheckURL,
		"https://ghfast.top/" + updateCheckURL,
		updateCheckURL,
	}
	if !reflect.DeepEqual(updateCheckURLs, wantChecks) {
		t.Fatalf("updateCheckURLs = %#v, want %#v", updateCheckURLs, wantChecks)
	}

	assetURL := updateRepoURL + "/releases/download/v1.11.2/hermes-dock-v1.11.2-linux-amd64.tar.gz"
	wantDownloads := []string{
		"https://gh-proxy.com/" + assetURL,
		"https://ghfast.top/" + assetURL,
		assetURL,
	}
	if got := updateDownloadCandidates(assetURL); !reflect.DeepEqual(got, wantDownloads) {
		t.Fatalf("updateDownloadCandidates() = %#v, want %#v", got, wantDownloads)
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

func TestPostUpdateTaskActive(t *testing.T) {
	for _, state := range []string{postUpdateStatePending, postUpdateStateWaiting, postUpdateStateSyncing, postUpdateStateApplying} {
		if !postUpdateTaskActive(state) {
			t.Fatalf("post-update state %q should be active", state)
		}
	}
	for _, state := range []string{"", postUpdateStateSucceeded, postUpdateStateFailed} {
		if postUpdateTaskActive(state) {
			t.Fatalf("post-update state %q should be terminal", state)
		}
	}
}

func TestShouldAutoApplyPostUpdate(t *testing.T) {
	if !shouldAutoApplyPostUpdate(true, true, "running") {
		t.Fatal("changed content should apply while Hermes remained running")
	}
	for _, test := range []struct {
		changed       bool
		wasRunning    bool
		currentStatus string
	}{
		{changed: false, wasRunning: true, currentStatus: "running"},
		{changed: true, wasRunning: false, currentStatus: "running"},
		{changed: true, wasRunning: true, currentStatus: "stopped"},
		{changed: true, wasRunning: true, currentStatus: "missing"},
		{changed: true, wasRunning: true, currentStatus: "unknown"},
	} {
		if shouldAutoApplyPostUpdate(test.changed, test.wasRunning, test.currentStatus) {
			t.Fatalf("unexpected auto apply for %+v", test)
		}
	}
}
