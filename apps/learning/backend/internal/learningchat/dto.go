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
	Available bool `json:"available"`
	// Teacher is whether the agent plays the language teacher in this chat.
	Teacher  bool              `json:"teacher"`
	Messages []messageResponse `json:"messages"`
	Budget   *budgetResponse   `json:"budget"`
	// Daily is what the user may still spend today across all their chats.
	Daily *budgetResponse `json:"daily"`
}

// doneEvent closes a stream: the stored question and answer, with the ids and
// times the history shows, and the budget with the turn counted.
type doneEvent struct {
	Question messageResponse `json:"question"`
	Answer   messageResponse `json:"answer"`
	Budget   budgetResponse  `json:"budget"`
	Daily    budgetResponse  `json:"daily"`
}

func dailyDTO(u commands.Usage) budgetResponse {
	resets := u.ResetsAt.UTC().Format(time.RFC3339)

	return budgetResponse{Limit: u.Limit, Used: u.Used, ResetsAt: &resets}
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
	out := chatResponse{Available: chat.Available, Teacher: chat.Teacher, Messages: make([]messageResponse, len(chat.Messages))}
	for i, row := range chat.Messages {
		out.Messages[i] = messageDTO(row)
	}

	if chat.Budget != nil {
		budget := budgetDTO(*chat.Budget)
		out.Budget = &budget
	}

	if chat.Daily != nil {
		daily := dailyDTO(*chat.Daily)
		out.Daily = &daily
	}

	return out
}

func doneDTO(reply commands.Reply) doneEvent {
	return doneEvent{Question: messageDTO(reply.Question), Answer: messageDTO(reply.Answer), Budget: budgetDTO(reply.Budget), Daily: dailyDTO(reply.Daily)}
}

// dialogResponse is a closed conversation in the history of a topic. Title is
// how it began; the messages are read separately.
type dialogResponse struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	MessageCount int64  `json:"messageCount"`
	StartedAt    string `json:"startedAt"`
	ArchivedAt   string `json:"archivedAt"`
}

type transcriptResponse struct {
	dialogResponse
	Messages []messageResponse `json:"messages"`
}

func dialogDTO(d queries.Dialog) dialogResponse {
	return dialogResponse{
		ID:           d.ID.String(),
		Title:        d.FirstMessage,
		MessageCount: d.MessageCount,
		StartedAt:    d.StartedAt.UTC().Format(time.RFC3339),
		ArchivedAt:   d.ArchivedAt.UTC().Format(time.RFC3339),
	}
}

func dialogsDTO(dialogs []queries.Dialog) []dialogResponse {
	out := make([]dialogResponse, len(dialogs))
	for i, d := range dialogs {
		out[i] = dialogDTO(d)
	}

	return out
}

func transcriptDTO(t queries.Transcript) transcriptResponse {
	out := transcriptResponse{dialogResponse: dialogDTO(t.Dialog), Messages: make([]messageResponse, len(t.Messages))}
	for i, row := range t.Messages {
		out.Messages[i] = messageDTO(row)
	}

	return out
}
