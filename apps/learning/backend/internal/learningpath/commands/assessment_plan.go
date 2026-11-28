package commands

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/learningpath/assessment"
	"learning_app/backend/internal/learningpath/curriculum"
	"learning_app/backend/internal/learningpath/placement"
)

func analyzeScreening(test db.LearningPathTest, prefs Preferences) (program, error) {
	var result assessment.Result
	if err := json.Unmarshal(test.AssessmentResult, &result); err != nil {
		return program{}, err
	}
	lang, explain := test.Language, test.AssessmentExplanationLanguage.String
	library := curriculum.UpTo(lang, explain, placement.B2)
	if len(library) == 0 {
		return program{}, fmt.Errorf("assessment library unavailable")
	}
	allowed := make(map[string]curriculum.Topic, len(library))
	for _, topic := range library {
		allowed[topic.ID] = topic
	}
	demonstrated := make(map[string]bool)
	for _, item := range result.DemonstratedStrengths {
		demonstrated[item.ItemKey] = true
	}
	selected := make(map[string]bool)
	plan := catalogPlan{Summary: screeningPlanSummary(explain)}
	// Bound the starting plan to roughly one week's study. The current path
	// still schedules the existing per-topic practice and due-review activities.
	limit := 10
	if prefs.HoursPerWeek > 0 {
		limit = min(14, max(6, prefs.HoursPerWeek*2))
	}
	var add func(string, int)
	add = func(id string, ceiling int) {
		if _, ok := allowed[id]; !ok || selected[id] || demonstrated[id] || len(plan.Topics) >= ceiling {
			return
		}
		selected[id] = true
		for _, req := range assessment.Prerequisites(lang, id) {
			add(req, ceiling)
		}
		if len(plan.Topics) < ceiling {
			plan.Topics = append(plan.Topics, catalogPick{ID: id})
		}
	}
	// Always leave room to start practising unassessed skills.
	for _, item := range result.RecommendedLearningObjectives {
		add(item.ItemKey, limit-2)
	}
	for _, item := range result.RecommendedKnowledgeChecks {
		add(item.ItemKey, limit-2)
	}
	if len(plan.Topics) == 0 {
		for _, topic := range library {
			if topic.Level == test.LevelReached && topic.Skill == "grammar" {
				add(topic.ID, 3)
			}
		}
	}

	unknown := make(map[string]bool)
	for _, skill := range result.UnassessedSkills {
		unknown[skill] = true
	}
	var practice []curriculum.Topic
	for _, topic := range library {
		if topic.Skill != "grammar" && unknown[topic.Skill] &&
			placement.Index(placement.Level(topic.Level)) <= placement.Index(placement.Level(test.LevelReached)) {
			practice = append(practice, topic)
		}
	}
	slices.SortStableFunc(practice, func(a, b curriculum.Topic) int {
		return cmp.Compare(preferenceScore(b, prefs), preferenceScore(a, prefs))
	})
	skills := make(map[string]bool)
	for _, topic := range practice {
		if !skills[topic.Skill] {
			add(topic.ID, limit)
			skills[topic.Skill] = true
		}
	}
	for _, topic := range practice {
		add(topic.ID, limit)
	}

	vocabulary := func(id string) []curriculum.Word {
		topic, _ := curriculum.Find(lang, test.TranslationLanguage, id)
		return topic.Vocabulary
	}
	return programFrom(plan, allowed, vocabulary), nil
}

func screeningPlanSummary(language string) string {
	switch language {
	case "es":
		return "El plan comienza con las reglas que conviene practicar o comprobar y añade actividades para destrezas aún sin evaluar."
	case "ru":
		return "В начале программы — правила для практики и проверки, а также задания по навыкам, которые ещё не оценивались."
	default:
		return "Your path starts with rules to practise or check and includes activities for skills the screening did not assess."
	}
}

func preferenceScore(topic curriculum.Topic, prefs Preferences) int {
	goal := strings.ToLower(prefs.Goal + " " + prefs.Interests)
	score := 0
	if strings.Contains(goal, topic.Skill) {
		score++
	}
	for _, tag := range topic.Tags {
		if tag != "" && strings.Contains(goal, strings.ToLower(tag)) {
			score++
		}
	}
	return score
}
