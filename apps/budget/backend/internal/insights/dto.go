package insights

import (
	"github.com/google/uuid"

	"budget_me/backend/internal/insights/queries"
)

// insightsResponse — новый контракт экрана AI: карточки-правила, честная цифра
// перерасхода (раньше на фронте был хардкод «147 Br») и текст совета из кэша
// (null, если ещё не написан или ключ модели не задан).
type insightsResponse struct {
	Insights       []insightResponse `json:"insights"`
	OverspendMinor int64             `json:"overspendMinor"`
	Advice         *string           `json:"advice"`
}

// insightResponse — одна карточка. Все числа приходят С БЭКА: раньше они были
// вписаны прямо в тексты переводов (и взяты с потолка).
type insightResponse struct {
	Kind         string `json:"kind"` // overspend | noLimit | onTrack
	Tone         string `json:"tone"`
	CategoryID   string `json:"categoryId,omitempty"`
	SpentMinor   int64  `json:"spentMinor,omitempty"`
	LimitMinor   int64  `json:"limitMinor,omitempty"`
	SuggestMinor int64  `json:"suggestMinor,omitempty"`
}

func toInsightResponse(it queries.Insight) insightResponse {
	resp := insightResponse{
		Kind:         it.Kind,
		Tone:         it.Tone,
		SpentMinor:   it.SpentMinor,
		LimitMinor:   it.LimitMinor,
		SuggestMinor: it.SuggestMinor,
	}

	if it.CategoryID != uuid.Nil {
		resp.CategoryID = it.CategoryID.String()
	}

	return resp
}
