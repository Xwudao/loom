package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillsCmdInstallsEmbeddedSkillInNearestProject(t *testing.T) {
	project := t.TempDir()
	skills := filepath.Join(project, ".agents", "skills")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(project, "internal", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := skillsCmd(nil); err != nil {
		t.Fatalf("skillsCmd: %v", err)
	}

	skillPath := filepath.Join(skills, "loom", "SKILL.md")
	skill, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(skill, loomSkill) {
		t.Errorf("%s does not contain the embedded skill", skillPath)
	}
	if !strings.Contains(string(skill), "references/wire-migration.md") {
		t.Error("skill does not link to the Wire migration reference")
	}

	migrationPath := filepath.Join(skills, "loom", "references", "wire-migration.md")
	migration, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(migration, wireMigrationSkill) {
		t.Errorf("%s does not contain the embedded migration reference", migrationPath)
	}

	if err := skillsCmd(nil); err != nil {
		t.Fatalf("second skillsCmd: %v", err)
	}
}

func TestFindSkillsDirRequiresProjectDirectory(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	_, err = findSkillsDir()
	if err == nil || !strings.Contains(err.Error(), "no .agents/skills directory found") {
		t.Errorf("findSkillsDir() error = %v, want missing directory error", err)
	}
}
