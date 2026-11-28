package learningpath

// startTestRequest opens a placement or a final test.
type startTestRequest struct {
	Purpose             string `json:"purpose"`
	StartPoint          string `json:"startPoint"`
	Language            string `json:"language"`
	TranslationLanguage string `json:"translationLanguage"`
	// Level starts a placement above A1 (only the rung after the level the
	// learner has just confirmed in a final test is accepted), or, on
	// POST /path/levels, is the already reached level to build a program at.
	Level string `json:"level"`
	// SelfReported, on POST /path/levels, makes Level a level the learner only
	// claims: it needs no history and is recorded as self-reported.
	SelfReported bool `json:"selfReported"`
}

// answersRequest carries one batch of test answers.
type answersRequest struct {
	Answers []struct {
		Key      string `json:"key"`
		Answer   string `json:"answer"`
		Skip     bool   `json:"skip"`
		Assisted bool   `json:"assisted"`
	} `json:"answers"`
}

// attemptRequest carries one exercise answer.
type attemptRequest struct {
	Answer string `json:"answer"`
	// ActivityID, when set, makes the answer one to a due review.
	ActivityID string `json:"activityId"`
}

// analyzeRequest carries the survey answers the program is built around.
type analyzeRequest struct {
	Interests    string `json:"interests"`
	Variant      string `json:"variant"`
	Goal         string `json:"goal"`
	HoursPerWeek int    `json:"hoursPerWeek"`
}

// goalRequest sets how many hours a week the learner means to study.
type goalRequest struct {
	Language     string `json:"language"`
	HoursPerWeek int    `json:"hoursPerWeek"`
}
