package learningpath

import (
	"context"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/httpx"
	learningcommands "learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/learningpath/commands"
	"learning_app/backend/internal/learningpath/curriculum"
)

type practiceExampleResponse struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

// practiceItemResponse is a line of a drill without its key.
type practiceItemResponse struct {
	ID      string   `json:"id"`
	Text    string   `json:"text"`
	Hint    string   `json:"hint"`
	Options []string `json:"options"`
}

type practiceBlockResponse struct {
	ID          string                 `json:"id"`
	Kind        string                 `json:"kind"`
	Instruction string                 `json:"instruction"`
	Items       []practiceItemResponse `json:"items"`
}

type practiceResponse struct {
	Examples []practiceExampleResponse `json:"examples"`
	Blocks   []practiceBlockResponse   `json:"blocks"`
}

// practiceDTO renders a sitting for the page of a rule: its sentences and its
// drills, with every key left out.
func practiceDTO(p commands.Practice) practiceResponse {
	out := practiceResponse{Examples: make([]practiceExampleResponse, len(p.Examples)), Blocks: make([]practiceBlockResponse, len(p.Blocks))}

	for i, e := range p.Examples {
		out.Examples[i] = practiceExampleResponse{ID: e.ID, Text: e.Text, Translation: e.Translation}
	}

	for i, b := range p.Blocks {
		block := practiceBlockResponse{ID: b.ID, Kind: b.Kind, Instruction: b.Instruction, Items: make([]practiceItemResponse, len(b.Items))}

		for j, it := range b.Items {
			block.Items[j] = practiceItemResponse{ID: it.ID, Text: it.Text, Hint: it.Hint, Options: it.Options}
		}

		out.Blocks[i] = block
	}

	return out
}

type checkRequest struct {
	Block  string `json:"block"`
	Item   string `json:"item"`
	Answer string `json:"answer"`
}

type checkResponse struct {
	Correct bool   `json:"correct"`
	Key     string `json:"key"`
	Note    string `json:"note"`
}

// ruleSubject is the rule of the URL, as far as the learner's teacher covers it.
func (c *Controller) ruleSubject(ctx echo.Context) (commands.RuleSubject, error) {
	language, native, err := c.ruleBook(ctx, ctx.Param("id"))
	if err != nil {
		return commands.RuleSubject{}, err
	}

	rule, ok := curriculum.Find(language, native, ctx.Param("id"))
	if !ok || rule.Extra == nil {
		return commands.RuleSubject{}, apperr.NewNotFound("rule not found")
	}

	return commands.RuleSubject{
		ID: rule.ID, Language: language, Translation: native,
		LanguageName: learningcommands.LanguageName(language), TranslationName: learningcommands.LanguageName(native),
		Level: rule.Level, Title: rule.Title, Theory: rule.Theory, Forms: rule.Extra.Forms,
	}, nil
}

// practice reads the sitting the learner has on a rule, and so tells that they
// are back on its page.
func (c *Controller) practice(ctx echo.Context) error {
	rule, err := c.ruleSubject(ctx)
	if err != nil {
		return err
	}

	p, err := c.practices.Open(ctx.Request().Context(), userOf(ctx), rule.ID)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, practiceDTO(p))
}

// leavePractice tells that the learner has gone from the page of a rule.
func (c *Controller) leavePractice(ctx echo.Context) error {
	rule, err := c.ruleSubject(ctx)
	if err != nil {
		return err
	}

	if err := c.practices.Leave(ctx.Request().Context(), userOf(ctx), rule.ID); err != nil {
		return err
	}

	return httpx.OK(ctx, map[string]bool{"ok": true})
}

// writeExamples has the agent write more sentences for a rule.
func (c *Controller) writeExamples(ctx echo.Context) error {
	return c.write(ctx, c.practices.AddExamples)
}

// writeExercises has the agent write the drills of a rule.
func (c *Controller) writeExercises(ctx echo.Context) error {
	return c.write(ctx, c.practices.WriteExercises)
}

// write runs one of the two things that spend the learner's key.
func (c *Controller) write(ctx echo.Context, do func(context.Context, uuid.UUID, commands.RuleSubject) (commands.Practice, error)) error {
	rule, err := c.ruleSubject(ctx)
	if err != nil {
		return err
	}

	release, wait, ok := c.limit.Acquire(userOf(ctx))
	if !ok {
		return tooManyRequests(ctx, wait)
	}

	defer release()

	p, err := do(ctx.Request().Context(), userOf(ctx), rule)
	if err = agentError(err); err != nil {
		return err
	}

	return httpx.OK(ctx, practiceDTO(p))
}

// checkPractice grades one line of a drill by its key.
func (c *Controller) checkPractice(ctx echo.Context) error {
	rule, err := c.ruleSubject(ctx)
	if err != nil {
		return err
	}

	var req checkRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	result, err := c.practices.Check(ctx.Request().Context(), userOf(ctx), rule.ID, req.Block, req.Item, req.Answer)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, checkResponse{Correct: result.Correct, Key: result.Key, Note: result.Note})
}

// practiceKey gives the key of one line to a learner who gives up on it.
func (c *Controller) practiceKey(ctx echo.Context) error {
	rule, err := c.ruleSubject(ctx)
	if err != nil {
		return err
	}

	var req checkRequest
	if err := ctx.Bind(&req); err != nil {
		return apperr.NewInvalid("invalid body")
	}

	result, err := c.practices.Key(ctx.Request().Context(), userOf(ctx), rule.ID, req.Block, req.Item)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, checkResponse{Key: result.Key, Note: result.Note})
}
