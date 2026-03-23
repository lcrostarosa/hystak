package discovery

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/hystak/hystak/internal/model"
)

// ScanSkills discovers skill directories under <project>/.claude/skills/ (S-011).
// Each subdirectory containing SKILL.md is a candidate skill.
func ScanSkills(projectPath string) ([]model.SkillDef, error) {
	skillsDir := filepath.Join(projectPath, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var skills []model.SkillDef
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		_, statErr := os.Stat(skillFile)
		switch {
		case statErr == nil:
			skills = append(skills, model.SkillDef{
				Name:   e.Name(),
				Source: skillFile,
			})
		case errors.Is(statErr, fs.ErrNotExist):
			continue
		default:
			return nil, statErr
		}
	}

	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, nil
}
