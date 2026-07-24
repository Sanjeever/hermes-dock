package main

import (
	"os"
	"sync"
	"testing"
)

func TestUpdateStateSerializesConcurrentMutations(t *testing.T) {
	app := newTestApp(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := app.updateState(func(state *LauncherState) error {
				state.Backups = append(state.Backups, BackupRecord{ID: string(rune('a' + id))})
				return nil
			}); err != nil {
				t.Errorf("update state: %v", err)
			}
		}(i)
	}
	wg.Wait()

	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Backups) != 20 {
		t.Fatalf("backups = %d, want 20", len(state.Backups))
	}
}

func TestProviderAndProxySaveRejectCorruptStateBeforeWriting(t *testing.T) {
	for _, test := range []struct {
		name string
		path func(*App) string
		save func(*App) error
	}{
		{
			name: "provider",
			path: func(app *App) string { return app.defaultConfigPath() },
			save: func(app *App) error {
				providers, err := app.readProviderConfigForProfile(defaultProfileID)
				if err != nil {
					return err
				}
				return app.SaveProviderConfigForProfile(defaultProfileID, providers)
			},
		},
		{
			name: "proxy",
			path: func(app *App) string { return app.proxyPath() },
			save: func(app *App) error { return app.SaveProxySettings(defaultProxySettings()) },
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			path := test.path(app)
			before, err := os.ReadFile(path)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if err := os.WriteFile(app.statePath(), []byte("{"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := test.save(app); err == nil {
				t.Fatal("save should reject a corrupt state file")
			}
			after, err := os.ReadFile(path)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Fatal("managed file changed after state preflight failed")
			}
		})
	}
}
