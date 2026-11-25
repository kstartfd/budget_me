package learningpath

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/httpx"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningpath/commands"
	"learning_app/backend/internal/learningpath/queries"
	"learning_app/backend/internal/learningprofile"
)

// reviewExerciseResponse is an exercise asked for a review: the same as an
// exercise of a program but with no key and no rule, because a review is
// independent evidence of what the learner can do without them.
type reviewExerciseResponse struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Text    string   `json:"text"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

type topicRefResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type itemRefResponse struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Level string `json:"level"`
	Skill string `json:"skill"`
}

// activityResponse is one thing to do: its type, why it is offered, what it is for.
type activityResponse struct {
	ID        string                  `json:"id"`
	Type      string                  `json:"type"`
	Status    string                  `json:"status"`
	Reason    string                  `json:"reason"`
	Title     string                  `json:"title"`
	Goal      string                  `json:"goal"`
	Item      *itemRefResponse        `json:"item"`
	Topic     *topicRefResponse       `json:"topic"`
	State     string                  `json:"state"`
	Lapsed    bool                    `json:"lapsed"`
	DueAt     *time.Time              `json:"dueAt"`
	Exercise  *reviewExerciseResponse `json:"exercise"`
	CreatedAt time.Time               `json:"createdAt"`
}

func activityDTO(p commands.PlannedActivity) activityResponse {
	out := activityResponse{
		ID: ids.FromPg(p.Activity.ID).String(), Type: p.Activity.Type, Status: p.Activity.Status, Reason: p.Activity.Reason,
		Title: p.Title, Goal: p.Goal, State: p.State, Lapsed: p.Lapsed, DueAt: p.DueAt, CreatedAt: p.Activity.CreatedAt.Time,
	}

	if p.Item != nil {
		out.Item = &itemRefResponse{ID: ids.FromPg(p.Item.ID).String(), Key: p.Item.Key, Level: p.Item.CefrLevel, Skill: p.Item.Skill}
	}

	if p.Topic != nil {
		out.Topic = &topicRefResponse{ID: ids.FromPg(p.Topic.ID).String(), Title: p.Topic.Title}
	}

	if p.Exercise != nil {
		full := exerciseDTO(*p.Exercise)
		out.Exercise = &reviewExerciseResponse{ID: full.ID, Kind: full.Kind, Text: full.Text, Prompt: full.Prompt, Options: full.Options}
	}

	return out
}

func activitiesDTO(list []commands.PlannedActivity) []activityResponse {
	out := make([]activityResponse, len(list))
	for i, p := range list {
		out[i] = activityDTO(p)
	}

	return out
}

// activities lists what the learner should do now in a language, opening the
// reviews that have come due. Repeating the call opens nothing twice.
func (c *Controller) activities(ctx echo.Context) error {
	language := ctx.QueryParam("language")
	if language == "" {
		return apperr.NewInvalid("language is required")
	}

	planned, _, err := c.activity.Plan(ctx.Request().Context(), userOf(ctx), language)
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, activitiesDTO(planned))
}

type dashboardResponse struct {
	Language   string             `json:"language"`
	TeacherID  string             `json:"teacherId"`
	Items      dashboardItems     `json:"items"`
	Reviews    dashboardReviews   `json:"reviews"`
	Evidence   dashboardEvidence  `json:"evidence"`
	Activities []activityResponse `json:"activities"`
}

type dashboardItems struct {
	Tracked   int `json:"tracked"`
	Learning  int `json:"learning"`
	Practiced int `json:"practiced"`
	Mastered  int `json:"mastered"`
	Lapsed    int `json:"lapsed"`
}

type dashboardReviews struct {
	Due      int        `json:"due"`
	Overdue  int        `json:"overdue"`
	Upcoming int        `json:"upcoming"`
	NextDue  *time.Time `json:"nextDueAt"`
}

type dashboardEvidence struct {
	Total     int `json:"total"`
	Last7Days int `json:"last7Days"`
}

func dashboardDTO(language string, teacherID uuid.UUID, d queries.Dashboard, planned []commands.PlannedActivity) dashboardResponse {
	return dashboardResponse{
		Language:  language,
		TeacherID: teacherID.String(),
		Items: dashboardItems{
			Tracked: d.Tracked, Learning: d.Known["learning"], Practiced: d.Known["practiced"], Mastered: d.Known["mastered"], Lapsed: d.Lapsed,
		},
		Reviews:    dashboardReviews{Due: d.Reviews.Due, Overdue: d.Reviews.Overdue, Upcoming: d.Reviews.Upcoming, NextDue: d.Reviews.NextDue},
		Evidence:   dashboardEvidence{Total: d.EvidenceTotal, Last7Days: d.EvidenceRecent},
		Activities: activitiesDTO(planned),
	}
}

// dashboard is the minimal picture of the loop: what is known, what is due,
// what to do next. Every number is a count; a learner with no evidence sees zeros.
func (c *Controller) dashboard(ctx echo.Context) error {
	language := ctx.QueryParam("language")
	if language == "" {
		return apperr.NewInvalid("language is required")
	}

	reqCtx := ctx.Request().Context()

	planned, teacher, err := c.activity.Plan(reqCtx, userOf(ctx), language)
	if err != nil {
		return err
	}

	d, err := c.read.Dashboard(reqCtx, userOf(ctx), teacher, c.clock.Now(), learningprofile.Default())
	if err != nil {
		return err
	}

	ctx.Response().Header().Set("Cache-Control", "no-store")

	return httpx.OK(ctx, dashboardDTO(language, ids.FromPg(teacher.ID), d, planned))
}

// activityIDOf reads the optional activity an answer is given to.
func activityIDOf(raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.Nil, nil
	}

	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperr.NewInvalid("invalid activity id")
	}

	return id, nil
}
