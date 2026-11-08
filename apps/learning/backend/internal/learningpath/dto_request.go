package learningpath

// startTestRequest opens a placement or a final test.
type startTestRequest struct {
	Language            string `json:"language"`
	TranslationLanguage string `json:"translationLanguage"`
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
