package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/GehirnInc/crypt/sha512_crypt"
)

func TestDufsDefaultsCreateHashedSingleAccountConfig(t *testing.T) {
	app := newTestApp(t)
	settings := mustReadComposeSettings(t, app)
	if !settings.DufsEnabled || settings.DufsPort != defaultDufsPort || settings.DufsUsername != defaultDufsUsername {
		t.Fatalf("unexpected Dufs defaults: %+v", settings)
	}

	passwordHash, err := readDufsPasswordHash(app.dufsConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := sha512_crypt.New().Verify(passwordHash, []byte(defaultDufsPassword)); err != nil {
		t.Fatalf("default Dufs password does not match stored hash: %v", err)
	}
	config, err := os.ReadFile(app.dufsConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(config, []byte(":"+defaultDufsPassword+"@")) {
		t.Fatal("Dufs config contains the plaintext password")
	}
	state, err := os.ReadFile(app.statePath())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(state, []byte("dufsPassword")) {
		t.Fatal("launcher state must not persist the Dufs password field")
	}
}

func TestRenderComposeIncludesHardenedDufsSidecar(t *testing.T) {
	settings := defaultComposeSettings()
	compose := mustRenderCompose(t, settings, defaultProxySettings())
	for _, want := range []string{
		"image: " + defaultDufsImage,
		`user: "` + dufsContainerUser() + `"`,
		`0.0.0.0:9878:5000`,
		`./launcher/dufs/config.yaml:/etc/dufs.yaml:ro`,
		`read_only: true`,
		`no-new-privileges:true`,
		`cap_drop:`,
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("compose missing %q:\n%s", want, compose)
		}
	}
}

func TestSaveDufsPasswordMarksDufsOnlyApply(t *testing.T) {
	app := newTestApp(t)
	settings := mustReadComposeSettings(t, app)
	settings.DufsPassword = "new-password"
	if err := app.SaveComposeSettings(settings); err != nil {
		t.Fatal(err)
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.NeedsRebuild || !state.PendingDufsOnly {
		t.Fatalf("Dufs-only save should not require restarting Hermes: %+v", state)
	}
	if state.ComposeSettings.DufsPassword != "" {
		t.Fatal("Dufs plaintext password was persisted in launcher state")
	}
	passwordHash, err := readDufsPasswordHash(app.dufsConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := sha512_crypt.New().Verify(passwordHash, []byte("new-password")); err != nil {
		t.Fatalf("new Dufs password does not match stored hash: %v", err)
	}
}
