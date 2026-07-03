package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListProfileSkillsMarksCustomAndConflicts(t *testing.T) {
	app := newTestApp(t)
	writeTestSkill(t, app, "skills/custom/alpha", "local-alpha")
	writeTestSkill(t, app, "skills/custom/beta-one", "duplicate-skill")
	writeTestSkill(t, app, "skills/custom/beta-two", "duplicate-skill")

	state, err := app.ListProfileSkills()
	if err != nil {
		t.Fatal(err)
	}
	if state.BuiltinCount == 0 {
		t.Fatalf("expected bundled skills to be marked")
	}
	local := findSkill(state.Skills, "local-alpha", "skills/custom/alpha")
	if local == nil {
		t.Fatalf("custom skill not found")
	}
	if local.Builtin {
		t.Fatalf("custom skill marked builtin")
	}
	if state.ConflictCount != 1 {
		t.Fatalf("conflict count = %d, want 1", state.ConflictCount)
	}
	for _, path := range []string{"skills/custom/beta-one", "skills/custom/beta-two"} {
		item := findSkill(state.Skills, "duplicate-skill", path)
		if item == nil || !item.Conflict {
			t.Fatalf("duplicate skill at %s not marked as conflict: %+v", path, item)
		}
	}
}

func TestDeleteSkillBacksUpAndRejectsUnsafePath(t *testing.T) {
	app := newTestApp(t)
	writeTestSkill(t, app, "skills/custom/remove-me", "remove-me")

	if err := app.DeleteSkill("../data/skills/custom/remove-me"); err == nil {
		t.Fatalf("unsafe path should be rejected")
	}
	if err := app.DeleteSkill("skills/custom/remove-me"); err != nil {
		t.Fatal(err)
	}
	if fileExists(filepath.Join(app.currentProfileDataDir(), "skills", "custom", "remove-me")) {
		t.Fatalf("skill directory still exists")
	}
	state, err := app.readState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Backups) == 0 {
		t.Fatalf("delete did not record a backup")
	}
}

func writeTestSkill(t *testing.T, app *App, rel string, name string) {
	t.Helper()
	dir := filepath.Join(app.currentProfileDataDir(), filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Test skill\n---\n\n# Test\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func findSkill(skills []SkillSummary, name string, path string) *SkillSummary {
	for i := range skills {
		if skills[i].Name == name && skills[i].Path == path {
			return &skills[i]
		}
	}
	return nil
}
