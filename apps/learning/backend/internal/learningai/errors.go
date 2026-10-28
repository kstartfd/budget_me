package learningai

import (
	"net/http"

	"learning_app/backend/internal/httpx"
)

// What a caller can do about each is different, so each has its own business
// code: re-enter the key, try again later, or give up on this prompt.
var (
	agentRejected    = httpx.New(30001, http.StatusUnprocessableEntity, "the provider rejected the AI agent's API key")
	agentUnavailable = httpx.New(30002, http.StatusServiceUnavailable, "the AI agent is temporarily unavailable")
	agentFailed      = httpx.New(30003, http.StatusBadGateway, "the AI agent did not return an answer")
)
