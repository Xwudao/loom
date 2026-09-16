package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed skills/loom/SKILL.md
var loomSkill []byte

//go:embed skills/loom/references/wire-migration.md
var wireMigrationSkill []byte

func skillsCmd(args []string) error {
	fs := flag.NewFlagSet("loom skills", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("loom skills: unexpected argument %q", fs.Arg(0))
	}

	dir, err := findSkillsDir()
	if err != nil {
		return err
	}
	for _, file := range []struct {
		path    string
		content []byte
	}{
		{filepath.Join(dir, "loom", "SKILL.md"), loomSkill},
		{filepath.Join(dir, "loom", "references", "wire-migration.md"), wireMigrationSkill},
	} {
		changed, err := writeSkill(file.path, file.content)
		if err != nil {
			return err
		}
		if changed {
			fmt.Printf("loom: wrote %s\n", relPath(file.path))
		} else {
			fmt.Printf("loom: %s unchanged\n", relPath(file.path))
		}
	}
	return nil
}

// findSkillsDir finds the nearest project skill directory, allowing loom skills
// to be installed when the command is run from a subdirectory of the project.
func findSkillsDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("loom skills: get working directory: %w", err)
	}
	for {
		candidate := filepath.Join(dir, ".agents", "skills")
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("loom skills: inspect %s: %w", candidate, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("loom skills: no .agents/skills directory found; create one in the project first")
}

func writeSkill(path string, content []byte) (bool, error) {
	current, err := os.ReadFile(path)
	if err == nil && bytes.Equal(current, content) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("loom skills: read %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("loom skills: create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return false, fmt.Errorf("loom skills: write %s: %w", path, err)
	}
	return true, nil
}
