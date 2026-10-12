package httpx

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/token"
)

// LearningTokenAuthenticator returns the token's owner, whether it is
// capture-only, and the single topic it is bound to (uuid.Nil if unbound).
type LearningTokenAuthenticator func(context.Context, string) (uuid.UUID, bool, uuid.UUID, error)

func RequireLearningAuth(codec *token.Codec, authenticate LearningTokenAuthenticator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw, ok := strings.CutPrefix(c.Request().Header.Get(echo.HeaderAuthorization), "Bearer ")
			if !ok || raw == "" {
				return Unauthorized
			}

			var (
				userID  uuid.UUID
				topicID uuid.UUID
				err     error
			)

			if strings.HasPrefix(raw, "bml_") {
				var captureOnly bool

				userID, captureOnly, topicID, err = authenticate(c.Request().Context(), raw)
				if err != nil {
					return err
				}

				path, method := c.Path(), c.Request().Method
				captures := path == "/api/v1/learning/captures" || path == "/api/v1/learning/captures/:id"
				topics := path == "/api/v1/learning/topics" && method == http.MethodGet

				entries := (path == "/api/v1/learning/entries" || path == "/api/v1/learning/entries/:id") && method != http.MethodPost
				if !captures && !topics && (captureOnly || !entries) {
					return apperr.NewForbidden("token scope does not allow this operation")
				}
			} else {
				userID, err = codec.Parse(raw, time.Now().UTC())
				if err != nil {
					return Unauthorized
				}
			}

			req := c.Request()
			ctx := context.WithValue(req.Context(), userIDKey{}, userID)
			ctx = context.WithValue(ctx, learningTopicKey{}, topicID)
			c.SetRequest(req.WithContext(ctx))

			return next(c)
		}
	}
}
