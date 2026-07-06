package main

import "testing"

func TestMaskEnvironmentForWebKeepsBindingSignals(t *testing.T) {
	env := []EnvVar{
		{Key: "WEIXIN_ACCOUNT_ID", Value: "wx-account", Secret: false},
		{Key: "WEIXIN_TOKEN", Value: "wx-token", Secret: true},
		{Key: "WECOM_SECRET", Value: "", Secret: true},
	}

	masked := maskEnvironmentForWeb(env)

	if got := envValue(masked, "WEIXIN_ACCOUNT_ID"); got != "wx-account" {
		t.Fatalf("WEIXIN_ACCOUNT_ID = %q, want wx-account", got)
	}
	if got := envValue(masked, "WEIXIN_TOKEN"); got != "<redacted>" {
		t.Fatalf("WEIXIN_TOKEN = %q, want <redacted>", got)
	}
	if got := envValue(masked, "WECOM_SECRET"); got != "" {
		t.Fatalf("empty WECOM_SECRET = %q, want empty", got)
	}
}
