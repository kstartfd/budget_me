package assessment

import "fmt"

// SectionState is the persisted lifecycle projection needed by the shared mode
// coordinator. A missing section has not started yet.
type SectionState struct {
	Skill  AssessedSkill
	Status string
}

type ModeState struct {
	Status    string
	NextSkill AssessedSkill
}

// AggregateMode derives the overall lifecycle from independently progressing
// skill sections. It does not assign scores or infer a level from missing skills.
// At most one section may be active at a time, while completed sections remain
// available even when the learner pauses before the next one.
func AggregateMode(mode ModeBlueprint, sections []SectionState) (ModeState, error) {
	if err := mode.Validate(); err != nil {
		return ModeState{}, err
	}
	known := make(map[AssessedSkill]bool, len(mode.Sections))
	for _, section := range mode.Sections {
		known[section.Skill] = true
	}
	states := make(map[AssessedSkill]string, len(sections))
	var activeSkill AssessedSkill
	for _, section := range sections {
		if !known[section.Skill] || states[section.Skill] != "" || !validSectionStatus(section.Status) {
			return ModeState{}, fmt.Errorf("invalid section state for %q", section.Skill)
		}
		states[section.Skill] = section.Status
		if isActiveSection(section.Status) {
			if activeSkill != "" {
				return ModeState{}, fmt.Errorf("multiple active assessment sections")
			}
			activeSkill = section.Skill
		}
	}
	if activeSkill != "" {
		status := states[activeSkill]
		if status == Created {
			for skill, previous := range states {
				if skill != activeSkill && !isActiveSection(previous) {
					status = InProgress
					break
				}
			}
		}
		return ModeState{Status: status, NextSkill: activeSkill}, nil
	}
	var next AssessedSkill
	completed, failed, cancelled, terminal := 0, 0, 0, 0
	for _, section := range mode.Sections {
		status := states[section.Skill]
		if status == "" {
			if next == "" {
				next = section.Skill
			}
			continue
		}
		terminal++
		switch status {
		case Completed:
			completed++
		case Failed:
			failed++
		case Cancelled:
			cancelled++
		}
	}
	if next != "" {
		status := InProgress
		if terminal == 0 {
			status = Created
		}
		return ModeState{Status: status, NextSkill: next}, nil
	}
	count := len(mode.Sections)
	switch {
	case completed == count:
		return ModeState{Status: Completed}, nil
	case failed == count:
		return ModeState{Status: Failed}, nil
	case cancelled == count:
		return ModeState{Status: Cancelled}, nil
	default:
		return ModeState{Status: Partial}, nil
	}
}

func validSectionStatus(status string) bool {
	switch status {
	case Created, InProgress, Paused, Evaluating, Completed, Partial, Failed, Cancelled:
		return true
	default:
		return false
	}
}

func isActiveSection(status string) bool {
	switch status {
	case Created, InProgress, Paused, Evaluating:
		return true
	default:
		return false
	}
}
