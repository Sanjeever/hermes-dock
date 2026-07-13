package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateHostRPAMouseRequest(t *testing.T) {
	req := hostRPAMouseRequest{Action: "click", ExpectedWindowID: "win:1"}
	if err := validateHostRPAMouseRequest(&req); err != nil {
		t.Fatal(err)
	}
	if req.Button != "left" || req.Count != 1 {
		t.Fatalf("defaults = button %q count %d", req.Button, req.Count)
	}

	invalid := hostRPAMouseRequest{Action: "scroll", DY: 101, ExpectedWindowID: "win:1"}
	if err := validateHostRPAMouseRequest(&invalid); err == nil {
		t.Fatal("expected oversized scroll to fail")
	}
}

func TestValidateHostRPAKeyboardRequest(t *testing.T) {
	req := hostRPAKeyboardRequest{Action: "press", Key: "ENTER", ExpectedWindowID: "win:1"}
	if err := validateHostRPAKeyboardRequest(&req); err != nil {
		t.Fatal(err)
	}
	if req.Key != "enter" || req.Count != 1 {
		t.Fatalf("normalized request = key %q count %d", req.Key, req.Count)
	}

	tooLong := hostRPAKeyboardRequest{
		Action:           "type",
		Text:             strings.Repeat("a", hostRPAMaxText+1),
		ExpectedWindowID: "win:1",
	}
	if err := validateHostRPAKeyboardRequest(&tooLong); err == nil {
		t.Fatal("expected oversized text to fail")
	}
}

func TestHostRPAProfile(t *testing.T) {
	request := httptest.NewRequest("GET", "/v1/rpa/info", nil)
	if profile, err := hostRPAProfile(request); err != nil || profile != "default" {
		t.Fatalf("default profile = %q, %v", profile, err)
	}

	request.Header.Set("X-Hermes-Profile", "sales-2")
	if profile, err := hostRPAProfile(request); err != nil || profile != "sales-2" {
		t.Fatalf("named profile = %q, %v", profile, err)
	}

	request.Header.Set("X-Hermes-Profile", "Sales")
	if _, err := hostRPAProfile(request); err == nil {
		t.Fatal("expected invalid profile to fail")
	}
}
