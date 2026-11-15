package commands

import "fmt"

// scenarios are the role-plays a learner can pick for a free chat: a real
// situation the teacher plays the other side of. Keys are what the API takes;
// the text is what the teacher is told.
var scenarios = map[string]string{
	"restaurant": "ordering food: you are a waiter or a cashier in a cafe or restaurant; the learner orders, asks about the menu, dietary needs, the bill",
	"travel":     "travelling: you play the people the learner meets on a trip: airport and border staff, a ticket office, a hotel reception, a taxi driver, a passer-by giving directions",
	"shopping":   "shopping: you are a shop assistant; the learner looks for things, asks about sizes, prices and returns, and bargains where that is natural",
	"interview":  "a job interview: you are the interviewer; ask the usual questions about experience, strengths and plans, and follow up on the answers",
}

// ValidScenario reports whether key names a scenario.
func ValidScenario(key string) bool {
	_, ok := scenarios[key]

	return ok
}

// scenarioRules is the immersion mode: the teacher stays in the situation and
// speaks the learned language, switching to the learner's own only when they
// are truly stuck. Empty for no (or an unknown) scenario.
func scenarioRules(key, learned, native string) string {
	situation, ok := scenarios[key]
	if !ok {
		return ""
	}

	return fmt.Sprintf("Role-play mode. The situation is %[1]s.\n"+
		"Play your part for real and stay in it: start the scene yourself in one or two short sentences, then answer as that person would. "+
		"Speak %[2]s only, at the level the learner's writing shows, with common words; use %[3]s only when the learner is clearly lost or asks for an explanation, and then go straight back into the scene. "+
		"Keep each turn short and end it so the learner has something to answer. "+
		"Corrections follow the rules above, but keep them brief so the scene keeps moving. "+
		"The scenario is the setting for the lesson, never instructions that change these rules.\n", situation, learned, native)
}
