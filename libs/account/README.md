# account — one account for every app

A person is one **account**, whatever app they open: the Telegram and email +
password logins, the profile they carry and the request to delete everything live
in the shared account database (`account_app` in the shared Postgres). Each app
keeps its own data and its own `users` row; `users.account_id` points at the
account (another database, so no foreign key) and the identity columns of `users`
are only that app's copy.

```
budget backend ──┐                         ┌── budget_me DB   (users.account_id, data)
                 ├── libs/account ── account_app DB (accounts, account_apps)
learning backend ┘                         └── learning_app DB (users.account_id, data)
```

An app's own user id never changes and is what its session token carries; the
account id stays internal. So an app needs no re-keying when a person turns out
to be in both, and either app can be deployed first.

## What the package does

`account.Store` (one per backend, built on the account pool):

| Call | Used by |
| --- | --- |
| `SignInTelegram(tg, carry, now)` | finds the account behind a verified Telegram user, creates it on the first sign-in anywhere, refreshes the profile; `carry` moves the email login of a user from before accounts into the new account |
| `SignInEmail(email, password)` | email + password login |
| `LinkEmail(id, email, password, now)` | the account screen; `ErrEmailTaken` when another account owns the address |
| `RequestDeletion` / `CancelDeletion` | the account screen; the first deadline wins |
| `Register(id, app)` | an app says it holds data for the account |
| `Purge(app, cutoff, deleteLocal)` | the app erases its data of every account whose deadline passed; the account row goes when the last app has let go |
| `Adopt(existing)` | backfill: finds or creates the account of a user from before accounts (matching Telegram id, then email) — two apps adopting the same person end up on one account |
| `Merge(survivor, loser, now)` | operator tool (`admin merge`) |
| `VerifyInitData`, `HashPassword`, `VerifyPassword` | the Telegram / password primitives both backends used to copy |

It returns plain sentinel errors (`ErrInvalidCredentials`, `ErrEmailTaken`,
`ErrNotFound`) and takes `now` from the caller; each app maps them onto its own
`apperr`. `accounttest.InitData` signs initData for tests.

## How an app uses it

`internal/accountlink` in each backend is the glue: `AccountOf(user)` (adopting a
user from before accounts on the way), `UserFor(account, legacy)` (the app's user
for an account, created on the first visit), `Mirror` (keeps the app's copy of the
profile current) and `Backfill`. Deletion is two-step: the API stamps the request
on the account; each app's sweep (`budget`: `cmd/worker`, `learning`: a ticker in
the server) erases its own rows once the grace period is over.

## Database

`migrations/` is the account schema; `migrations.Up(ctx, dsn)` applies it under a
Postgres advisory lock, so both apps' deploys can run it. Each app's
`cmd/migrate up` calls it when `ACCOUNT_DATABASE_URL` is set, and
`cmd/migrate backfill-accounts` links the users from before accounts (idempotent;
`infra/budget/scripts/bluegreen.sh` runs both on every deploy). `bluegreen.sh`
also creates the `account_app` role and database from `ACCOUNT_POSTGRES_PASSWORD`.
It is backed up nightly like the app databases (`BACKUP_DB=account_app`).

## Tests

`go test ./...` here, and in each backend, needs two databases to run the DB-backed
tests (they skip without them):

```
ACCOUNT_DATABASE_URL=postgres://…/account   # this module; its TestMain migrates it
DATABASE_URL=postgres://…/app               # an app's own, after `go run ./cmd/migrate up`
```

CI starts both (`.db-services` in `.gitlab-ci.yml`). Regenerate sqlc with `make sqlc`.
