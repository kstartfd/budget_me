// Command worker is the deferred-job runner, invoked by cron on the box the
// same way as migrate and admin:
//
//	docker compose -f prod.yml --profile tools run --rm worker
//
// It does one sweep and exits — no daemon, nothing resident in the 1 GB box.
// Three jobs:
//
//	purge-accounts — erase this app's data of every account whose deletion
//	                 request is older than ACCOUNT_DELETION_GRACE (default 30
//	                 days). Everything the user owns follows via ON DELETE CASCADE;
//	                 the account itself goes once no app holds data for it.
//	                 Cancelling the request takes it out of the sweep, which is
//	                 what makes the deletion undoable until the deadline.
//
//	write-advice   — refresh each user's AI advice for the current month. This is
//	                 where the model is paid for: once per user per night, instead
//	                 of once per screen open. The API only generates on demand for
//	                 a user who has no advice at all yet (a new account should not
//	                 wait until tomorrow).
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	ident "budget_me/account"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	accountcommands "budget_me/backend/internal/account/commands"
	"budget_me/backend/internal/ai"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/config"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/insights/commands"
	"budget_me/backend/internal/insights/queries"
)

// betweenCalls paces the advice pass. The provider rate-limits (429 came back
// within seconds when generation ran unthrottled), and nothing here is in a
// hurry — it runs at 03:17.
const betweenCalls = 2 * time.Second

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		fail(err.Error())
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fail("connect: " + err.Error())
	}

	defer pool.Close()

	accountPool, err := pgxpool.New(ctx, cfg.AccountDatabaseURL)
	if err != nil {
		fail("connect account database: " + err.Error())
	}

	defer accountPool.Close()

	queries := db.New(pool)

	clk := clock.New()
	purgeAccounts(ctx, queries, ident.New(accountPool), clk, cfg.AccountDeletionGrace)
	writeAdvice(ctx, queries, clk, ai.New(cfg.AIAPIKey, cfg.AIModel, cfg.AIBaseURL))
}

func purgeAccounts(
	ctx context.Context,
	q *db.Queries,
	accounts *ident.Store,
	clk clock.Clock,
	grace time.Duration,
) {
	purged, err := accountcommands.NewPurgeHandler(q, accounts, clk, grace).Handle(ctx)
	if err != nil {
		fail("purge-accounts: " + err.Error())
	}

	fmt.Printf(
		"purge-accounts: grace=%s deleted=%d\n",
		grace, len(purged),
	)

	for _, id := range purged {
		fmt.Printf("  deleted account %s\n", id)
	}
}

// writeAdvice rewrites the current month's advice for everyone whose numbers
// have moved since it was last written. Unchanged numbers hash the same and cost
// nothing. A user's own zone decides which month is "current" for them — that is
// what users.timezone is for: no client is attached here to say.
func writeAdvice(ctx context.Context, q *db.Queries, clk clock.Clock, client *ai.Client) {
	if !client.Enabled() {
		fmt.Println("write-advice: no AI key, skipped")

		return
	}

	users, err := q.ListUsersForAdvice(ctx)
	if err != nil {
		fail("write-advice: " + err.Error())
	}

	facts := queries.NewFactsHandler(q)
	generate := commands.NewGenerateAdviceHandler(q, client, clk)

	var written, skipped, failed int

	for _, u := range users {
		userID := ids.FromPg(u.ID)

		// The nightly pass has no client attached, so the user's own settings say
		// which span is "current": their zone (already on the row) and their
		// monthStartDay decide the billing period the advice is written about.
		full, err := q.GetUserByID(ctx, u.ID)
		if err != nil {
			fmt.Printf("  %s: load user failed: %v\n", userID, err)

			failed++

			continue
		}

		from, to, month := queries.PeriodBounds(int(full.MonthStartDay), u.Timezone, clk.Now())

		monthFacts, err := facts.Handle(ctx, userID, month, from, to)
		if err != nil {
			fmt.Printf("  %s: facts failed: %v\n", userID, err)

			failed++

			continue
		}

		if monthFacts.Empty() {
			skipped++

			continue
		}

		cached, err := q.GetAdvice(ctx, db.GetAdviceParams{
			UserID: u.ID,
			Month:  monthFacts.Month,
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			fmt.Printf("  %s: read cache failed: %v\n", userID, err)

			failed++

			continue
		}

		if !commands.Stale(cached, monthFacts) {
			skipped++

			continue
		}

		if err := generate.Handle(ctx, commands.GenerateAdvice{
			UserID: userID,
			Facts:  monthFacts,
		}); err != nil {
			fmt.Printf("  %s: generation failed: %v\n", userID, err)

			failed++

			continue
		}

		written++

		time.Sleep(betweenCalls)
	}

	fmt.Printf(
		"write-advice: users=%d written=%d unchanged=%d failed=%d\n",
		len(users), written, skipped, failed,
	)
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "worker: "+msg)
	os.Exit(1)
}
