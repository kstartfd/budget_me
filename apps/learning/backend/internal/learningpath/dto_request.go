package learningpath

// startTestRequest opens a placement or a final test.
type startTestRequest struct {
	Language            string `json:"language"`
	TranslationLanguage string `json:"translationLanguage"`
	// Level starts a placement above A1. Only the rung after the level the
	// learner has just confirmed in a final test is accepted.
	Level string `json:"level"`
}

// answersRequest carries one batch of test answers.
type answersRequest struct {
	Answers []struct {
		Key    string `json:"key"`
		Answer string `json:"answer"`
	} `json:"answers"`
}

// attemptRequest carries one exercise answer.
type attemptRequest struct {
	Answer string `json:"answer"`
}

// analyzeRequest carries the survey answers the program is built around.
type analyzeRequest struct {
	Interests     string `json:"interests"`
	Variant       string `json:"variant"`
	Goal          string `json:"goal"`
	MinutesPerDay int    `json:"minutesPerDay"`
	DaysPerWeek   int    `json:"daysPerWeek"`
}
