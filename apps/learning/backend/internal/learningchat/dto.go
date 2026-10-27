package learningchat

import (
	"time"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/learningchat/commands"
	"learning_app/backend/internal/learningchat/queries"
)

type sendRequest struct {
	Text string `json:"text"`
}

type messageResponse struct {
	ID        int64  `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

// budgetResponse is what the chat may still spend: tokens used of the limit in
// the current window, which ends at resetsAt (null while nothing is spent).
type budgetResponse struct {
	Limit    int64   `json:"limit"`
	Used     int64   `json:"used"`
	ResetsAt *string `json:"resetsAt"`
}

type chatResponse struct {
	Available bool              `json:"available"`
	Messages  []messageResponse `json:"messages"`
	Budget    *budgetResponse   `json:"budget"`
}

// doneEvent closes a stream: the stored question and answer, with the ids and
// times the history shows, and the budget with the turn counted.
type doneEvent struct {
	Question messageResponse `json:"question"`
	Answer   messageResponse `json:"answer"`
	Budget   budgetResponse  `json:"budget"`
}

type deltaEvent struct {
	Text string `json:"text"`
}

type errorEvent struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func messageDTO(row db.LearningChatMessage) messageResponse {
	return messageResponse{ID: row.ID, Role: string(row.Role), Content: row.Content, CreatedAt: row.CreatedAt.Time.UTC().Format(time.RFC3339)}
}

func budgetDTO(b commands.Budget) budgetResponse {
	out := budgetResponse{Limit: b.Limit, Used: b.Used}

	if b.ResetsAt != nil {
		resets := b.ResetsAt.UTC().Format(time.RFC3339)
		out.ResetsAt = &resets
	}

	return out
}

func chatDTO(chat queries.Chat) chatResponse {
	out := chatResponse{Available: chat.Available, Messages: make([]messageResponse, len(chat.Messages))}
	for i, row := range chat.Messages {
		out.Messages[i] = messageDTO(row)
	}

	if chat.Budget != nil {
		budget := budgetDTO(*chat.Budget)
		out.Budget = &budget
	}

	return out
}

func doneDTO(reply commands.Reply) doneEvent {
	return doneEvent{Question: messageDTO(reply.Question), Answer: messageDTO(reply.Answer), Budget: budgetDTO(reply.Budget)}
}
