package commands

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
	"learning_app/backend/internal/secretbox"
)

const (
	maxQuestionRunes = 2000
	maxSummaryRunes  = 2000
	// chatLease outlives chatTimeout plus a compaction, so a chat is only ever
	// busy for longer than that if the server died holding it.
	chatLease      = 3 * time.Minute
	chatTimeout    = 90 * time.Second
	compactTimeout = 40 * time.Second
)

// ErrBusy: an answer or a compaction of this chat is still running.
var ErrBusy = apperr.NewConflict("the previous answer is still being written")

// Agent is the part of llm.Client a chat needs; tests swap in a fake.
type Agent interface {
	Chat(ctx context.Context, req llm.ChatRequest, onDelta func(string)) (llm.ChatResult, error)
	Complete(ctx context.Context, req llm.Request) (string, error)
}

// Send asks one question in a chat: the chat of an entry, or, with no EntryID,
// the free conversation of a topic.
type Send struct {
	UserID  uuid.UUID
	EntryID uuid.UUID
	TopicID uuid.UUID
	// PathTopicID is a topic of the learner's program: its own chat with the
	// teacher, about what that topic teaches.
	PathTopicID uuid.UUID
	Text        string
}

type Reply struct {
	Question db.LearningChatMessage
	Answer   db.LearningChatMessage
	// Budget is what the chat may still spend, with this turn counted.
	Budget Budget
	// Daily is what the user may still spend today across all chats.
	Daily Usage
}

// Stream receives the answer while it is written. Delta gets each piece; Done
// gets the stored messages once they are saved, before the history is folded,
// so the learner is never kept waiting for housekeeping. Either may be nil.
type Stream struct {
	Delta func(string)
	Done  func(Reply)
}

type SendHandler struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	clock clock.Clock
	vault secretbox.Box
	agent Agent
}

func NewSendHandler(pool *pgxpool.Pool, q *db.Queries, clk clock.Clock, vault secretbox.Box, agent Agent) SendHandler {
	return SendHandler{pool: pool, q: q, clock: clk, vault: vault, agent: agent}
}

// chatAgent is what a chat needs from the user's settings: the stored choice
// and the provider's own way of keeping a chat cheap.
type chatAgent struct {
	settings db.LearningAiSetting
	profile  llm.ChatProfile
	key      string
	// fingerprint names the key without revealing it: the daily limit of a key
	// is counted across every account that uses it.
	fingerprint string
}

// agentFor fails with a message the learner can act on when there is no agent
// to talk to: none chosen, a provider chat is not built for, or an unreadable key.
func (h SendHandler) agentFor(ctx context.Context, userID uuid.UUID) (chatAgent, error) {
	settings, err := h.q.GetLearningAISettings(ctx, ids.ToPg(userID))
	if db.IsNotFound(err) {
		return chatAgent{}, apperr.NewInvalid("chat needs an AI agent: set one up in settings")
	}

	if err != nil {
		return chatAgent{}, err
	}

	profile, ok := llm.ChatProfileOf(llm.Provider(settings.Provider))
	if !ok {
		return chatAgent{}, apperr.NewInvalid("chat is not available for this AI provider yet")
	}

	key, err := h.vault.Open(settings.KeyCiphertext, userID[:])
	if err != nil {
		slog.Error("learning ai key cannot be decrypted", "user_id", userID, "err", err)

		return chatAgent{}, apperr.NewInvalid("the saved API key cannot be read: enter it again in settings")
	}

	return chatAgent{settings: settings, profile: profile, key: string(key), fingerprint: h.vault.Fingerprint(key)}, nil
}

// target is what a chat is about: a topic, and the entry saved in it unless the
// chat is the free conversation of the topic.
type target struct {
	topic db.LearningTopic
	entry *db.LearningEntry
	// lesson is set for the chat of a program topic, which has no notebook.
	lesson *lessonSubject
}

// lessonSubject is a program topic and the program it belongs to.
type lessonSubject struct {
	topic db.LearningPathTopic
	path  db.LearningPath
}

// find resolves whose chat this is before anything else is looked at: whose
// it is must not depend on what the caller has set up.
func (h SendHandler) find(ctx context.Context, cmd Send) (target, error) {
	if cmd.PathTopicID != uuid.Nil {
		subject, err := h.lessonSubject(ctx, cmd.UserID, cmd.PathTopicID)

		return target{lesson: subject}, err
	}

	if cmd.EntryID == uuid.Nil {
		topic, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(cmd.TopicID), UserID: ids.ToPg(cmd.UserID)})
		if db.IsNotFound(err) {
			return target{}, apperr.NewNotFound("topic not found")
		}

		return target{topic: topic}, err
	}

	entry, err := h.q.GetLearningEntry(ctx, db.GetLearningEntryParams{ID: ids.ToPg(cmd.EntryID), UserID: ids.ToPg(cmd.UserID)})
	if db.IsNotFound(err) {
		return target{}, apperr.NewNotFound("entry not found")
	}

	if err != nil {
		return target{}, err
	}

	topic, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: entry.TopicID, UserID: entry.UserID})

	return target{topic: topic, entry: &entry}, err
}

// chat is created with the first question.
func (h SendHandler) ensureChat(ctx context.Context, t target, now time.Time) (db.LearningChat, error) {
	if t.lesson != nil {
		return h.q.EnsureLearningChatForPathTopic(ctx, db.EnsureLearningChatForPathTopicParams{ID: ids.ToPg(ids.New()), UserID: t.lesson.path.UserID, PathTopicID: t.lesson.topic.ID, CreatedAt: db.Time(now)})
	}

	if t.entry == nil {
		return h.q.EnsureOpenTopicChat(ctx, db.EnsureOpenTopicChatParams{ID: ids.ToPg(ids.New()), UserID: t.topic.UserID, TopicID: t.topic.ID, CreatedAt: db.Time(now)})
	}

	return h.q.EnsureLearningChat(ctx, db.EnsureLearningChatParams{ID: ids.ToPg(ids.New()), UserID: t.entry.UserID, EntryID: t.entry.ID, CreatedAt: db.Time(now)})
}

// system is what the model is told before the conversation. The free chat is
// always the teacher; a chat about an entry follows the learner's setting.
func (t target) system(summary string, teacher bool) string {
	if t.lesson != nil {
		return lessonSystem(t.lesson.path, t.lesson.topic, summary)
	}

	if t.entry == nil {
		return freeSystem(t.topic, summary)
	}

	return chatSystem(t.topic, *t.entry, summary, teacher)
}

// Handle answers one question of a chat. Nothing is stored unless
// the answer arrived whole, so a failed turn leaves the chat as it was and the
// learner can send the question again. The provider call is not tied to the
// request: a learner who leaves mid-answer still finds it there afterwards.
func (h SendHandler) Handle(ctx context.Context, cmd Send, stream Stream) error {
	question := strings.TrimSpace(cmd.Text)
	if question == "" || utf8.RuneCountInString(question) > maxQuestionRunes {
		return apperr.NewInvalid("the message must be 1 to 2000 characters")
	}

	subject, err := h.find(ctx, cmd)
	if err != nil {
		return err
	}

	agent, err := h.agentFor(ctx, cmd.UserID)
	if err != nil {
		return err
	}

	now := h.clock.Now()

	chat, err := h.ensureChat(ctx, subject, now)
	if err != nil {
		return err
	}

	chat, err = h.lock(ctx, chat, now)
	if err != nil {
		return err
	}

	// The lease is released even when the learner is gone, so it is not
	// bound to the request.
	work := context.WithoutCancel(ctx)
	defer func() { _ = h.q.UnlockLearningChat(work, chat.ID) }()

	// Read under the lease, so two questions cannot both slip in below the limit.
	budget := BudgetOf(chat, agent.profile, now)
	if budget.Exhausted() {
		return &BudgetError{ResetsAt: *budget.ResetsAt}
	}

	// The day's limits come after the chat's, so a refusal names the nearest end.
	daily, err := LoadDaily(work, h.q, cmd.UserID, agent.fingerprint, agent.profile, now)
	if err != nil {
		return err
	}

	if daily.User.Exhausted() || daily.Key.Exhausted() {
		slog.Info("learning chat daily limit", "user_id", cmd.UserID, "user_used", daily.User.Used, "key_used", daily.Key.Used)

		return &DailyError{ResetsAt: daily.User.ResetsAt}
	}

	stored, err := h.q.ListLearningChatMessagesAfter(work, db.ListLearningChatMessagesAfterParams{ChatID: chat.ID, ID: chat.SummaryUpto})
	if err != nil {
		return err
	}

	callCtx, cancel := context.WithTimeout(work, chatTimeout)
	defer cancel()

	result, err := h.agent.Chat(callCtx, llm.ChatRequest{
		Provider: llm.Provider(agent.settings.Provider),
		Model:    agent.settings.Model,
		Key:      agent.key,
		System:   subject.system(chat.Summary, agent.settings.Teacher),
		Messages: history(stored, question),
	}, stream.Delta)
	if err != nil {
		return err
	}

	reply, err := h.save(work, chat, turn{
		question:    question,
		result:      result,
		window:      agent.profile.BudgetWindow,
		after:       budget.After(result.Context(), now, agent.profile.BudgetWindow),
		daily:       daily.User.After(result.Context()),
		user:        cmd.UserID,
		fingerprint: agent.fingerprint,
	})
	if err != nil {
		return err
	}

	slog.Info("learning chat turn", "chat_id", ids.FromPg(chat.ID), "prompt_tokens", result.PromptTokens, "cached_tokens", result.CacheHitTokens, "reply_tokens", result.CompletionTokens)

	if stream.Done != nil {
		stream.Done(reply)
	}

	if result.Context() >= agent.profile.CompactAt {
		h.compact(work, chat, agent)
	}

	return nil
}

// lock takes the chat for one turn; nobody else can answer or fold it until
// the lease ends or is released.
func (h SendHandler) lock(ctx context.Context, chat db.LearningChat, now time.Time) (db.LearningChat, error) {
	locked, err := h.q.LockLearningChat(ctx, db.LockLearningChatParams{ID: chat.ID, UserID: chat.UserID, BusyUntil: db.Time(now.Add(chatLease)), Now: db.Time(now)})
	if db.IsNotFound(err) {
		return db.LearningChat{}, ErrBusy
	}

	return locked, err
}

// turn is one finished exchange on its way into the database: what was asked
// and answered, what it cost, and the limits as they stand with it counted
// (for the reply to carry).
type turn struct {
	question    string
	result      llm.ChatResult
	window      time.Duration
	after       Budget
	daily       Usage
	user        uuid.UUID
	fingerprint string
}

// save stores the question and its answer together, and records how big the
// conversation has become and what the turn cost the chat, the user and the key.
func (h SendHandler) save(ctx context.Context, chat db.LearningChat, t turn) (Reply, error) {
	question, result, window, after := t.question, t.result, t.window, t.after

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return Reply{}, err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	q := h.q.WithTx(tx)
	at := h.clock.Now()
	now := db.Time(at)

	asked, err := q.InsertLearningChatMessage(ctx, db.InsertLearningChatMessageParams{ChatID: chat.ID, Role: db.LearningChatRoleUser, Content: question, CreatedAt: now})
	if err != nil {
		return Reply{}, err
	}

	answered, err := q.InsertLearningChatMessage(ctx, db.InsertLearningChatMessageParams{ChatID: chat.ID, Role: db.LearningChatRoleAssistant, Content: result.Text, CreatedAt: now})
	if err != nil {
		return Reply{}, err
	}

	err = q.RecordLearningChatTurn(ctx, db.RecordLearningChatTurnParams{
		ID:            chat.ID,
		ContextTokens: int32(result.Context()),
		Now:           now,
		WindowSecs:    window.Seconds(),
		Spent:         int64(result.Context()),
	})
	if err != nil {
		return Reply{}, err
	}

	if err := recordUsage(ctx, q, t.user, t.fingerprint, result.Context(), at); err != nil {
		return Reply{}, err
	}

	return Reply{Question: asked, Answer: answered, Budget: after, Daily: t.daily}, tx.Commit(ctx)
}

// compact folds the older messages into the summary with the provider's cheap
// summarizing call. It is housekeeping: when it fails the chat is merely sent
// in full again, and the next turn tries once more.
func (h SendHandler) compact(ctx context.Context, chat db.LearningChat, agent chatAgent) {
	ctx, cancel := context.WithTimeout(ctx, compactTimeout)
	defer cancel()

	stored, err := h.q.ListLearningChatMessagesAfter(ctx, db.ListLearningChatMessagesAfterParams{ChatID: chat.ID, ID: chat.SummaryUpto})
	if err != nil {
		slog.Warn("learning chat compaction", "chat_id", ids.FromPg(chat.ID), "err", err)

		return
	}

	fold := foldable(stored, agent.profile.KeepTail)
	if len(fold) == 0 {
		return
	}

	system, prompt := summaryPrompt(chat.Summary, fold)

	text, err := h.agent.Complete(ctx, llm.Request{
		Provider: llm.Provider(agent.settings.Provider),
		Model:    agent.profile.SummaryModel,
		Level:    agent.profile.SummaryLevel,
		Key:      agent.key,
		System:   system,
		Prompt:   prompt,
	})
	if err != nil {
		slog.Warn("learning chat compaction", "chat_id", ids.FromPg(chat.ID), "provider", agent.settings.Provider, "err", err)

		return
	}

	summary := strings.TrimSpace(text)
	if summary == "" {
		return
	}

	if utf8.RuneCountInString(summary) > maxSummaryRunes {
		summary = string([]rune(summary)[:maxSummaryRunes])
	}

	err = h.q.SaveLearningChatSummary(ctx, db.SaveLearningChatSummaryParams{ID: chat.ID, Summary: summary, SummaryUpto: fold[len(fold)-1].ID, UpdatedAt: db.Time(h.clock.Now())})
	if err != nil {
		slog.Warn("learning chat compaction", "chat_id", ids.FromPg(chat.ID), "err", err)
	}
}

// Clear forgets the whole conversation of an entry: messages and summary.
// Nothing to forget is not an error.
func (h SendHandler) Clear(ctx context.Context, userID, entryID uuid.UUID) error {
	chat, err := h.q.GetLearningChatByEntry(ctx, db.GetLearningChatByEntryParams{EntryID: ids.ToPg(entryID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return h.forget(ctx, chat)
}

// forget deletes the messages and the summary of a chat under its lease.
func (h SendHandler) forget(ctx context.Context, chat db.LearningChat) error {
	now := h.clock.Now()

	chat, err := h.lock(ctx, chat, now)
	if err != nil {
		return err
	}

	defer func() { _ = h.q.UnlockLearningChat(context.WithoutCancel(ctx), chat.ID) }()

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	q := h.q.WithTx(tx)

	if err := q.DeleteLearningChatMessages(ctx, chat.ID); err != nil {
		return err
	}

	if err := q.ResetLearningChat(ctx, db.ResetLearningChatParams{ID: chat.ID, UpdatedAt: db.Time(now)}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// lessonSubject loads a program topic the learner owns, with its program.
func (h SendHandler) lessonSubject(ctx context.Context, userID, topicID uuid.UUID) (*lessonSubject, error) {
	topic, err := h.q.GetLearningPathTopic(ctx, db.GetLearningPathTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return nil, apperr.NewNotFound("topic not found")
	}

	if err != nil {
		return nil, err
	}

	path, err := h.q.GetLearningPath(ctx, db.GetLearningPathParams{ID: topic.PathID, UserID: ids.ToPg(userID)})
	if err != nil {
		return nil, err
	}

	return &lessonSubject{topic: topic, path: path}, nil
}

// ClearLesson forgets the whole conversation of a program topic.
func (h SendHandler) ClearLesson(ctx context.Context, userID, topicID uuid.UUID) error {
	chat, err := h.q.GetLearningChatByPathTopic(ctx, db.GetLearningChatByPathTopicParams{PathTopicID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return h.forget(ctx, chat)
}
