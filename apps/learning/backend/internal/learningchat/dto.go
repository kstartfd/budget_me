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

type chatResponse struct {
	Available bool              `json:"available"`
	Messages  []messageResponse `json:"messages"`
}

// doneEvent closes a stream: the stored question and answer, with the ids and
// times the history shows.
type doneEvent struct {
	Question messageResponse `json:"question"`
	Answer   messageResponse `json:"answer"`
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

func chatDTO(chat queries.Chat) chatResponse {
	out := chatResponse{Available: chat.Available, Messages: make([]messageResponse, len(chat.Messages))}
	for i, row := range chat.Messages {
		out.Messages[i] = messageDTO(row)
	}

	return out
}

func doneDTO(reply commands.Reply) doneEvent {
	return doneEvent{Question: messageDTO(reply.Question), Answer: messageDTO(reply.Answer)}
}
