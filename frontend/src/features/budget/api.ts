import { z } from "zod"

import { readToken } from "@/features/auth"
import { readMonthStartDay, readToday } from "@/features/settings"
import {
  currentMonthKey,
  periodBoundsISO,
  periodEndISO,
  periodOpenKey,
  periodStartISO,
  shiftMonth,
} from "@/lib/month"
import {
  browserTimezone,
  dayEndInstant,
  dayStartInstant,
  toInstant,
} from "@/lib/time"

import {
  balanceSchema,
  budgetSchema,
  categorySchema,
  insightsSchema,
  planSchema,
  planSuggestionSchema,
  rateSourceSchema,
  transactionSchema,
  trendPointSchema,
} from "./schemas"
import type {
  AddTransactionInput,
  Balance,
  Budget,
  Category,
  CreateCategoryInput,
  DeleteCategoryInput,
  Insights,
  MonthSummary,
  Plan,
  PlanSuggestion,
  RateSource,
  SaveBudgetInput,
  Transaction,
  TrendPoint,
  UpdateCategoryInput,
} from "./schemas"

/**
 * Budget "API" — the typed fetcher surface the query/mutation hooks call. Every
 * fetcher is live against the Go backend; the mock data layer the app was built
 * on is gone.
 */

/** Bars on the analytics trend chart. */
const trendMonths = 6

/**
 * Envelope-aware fetch against the Go backend (same-origin /api/v1, proxied by
 * Next rewrites). Mirrors features/auth/api.ts — non-zero code becomes an
 * ApiError. Used by fetchers already swapped off the mock DB.
 */
class ApiError extends Error {
  constructor(
    message: string,
    readonly code: number,
    readonly status: number,
  ) {
    super(message)
  }
}

/**
 * True when a create failed because a personal category with that name already
 * exists (case-insensitive). The backend answers HTTP 409 with envelope code
 * 40001; the create flow surfaces this as a "name taken" toast and keeps the
 * sheet open so the user can rename instead of dropping their input.
 */
export function isDuplicateNameError(error: unknown): boolean {
  return (
    error instanceof ApiError && (error.status === 409 || error.code === 40001)
  )
}

async function call<T>(
  path: string,
  init: RequestInit,
  parse: (data: unknown) => T,
): Promise<T> {
  const token = readToken()
  const res = await fetch(`/api/v1${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      // Protected routes (e.g. /transactions) need the session token; public
      // ones (e.g. /rates) simply ignore it. Skip the header when guest.
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init.headers,
    },
  })
  const body = (await res.json().catch(() => null)) as {
    code?: number
    message?: string
    data?: unknown
  } | null
  if (!body || body.code !== 0) {
    throw new ApiError(
      body?.message ?? res.statusText,
      body?.code ?? -1,
      res.status,
    )
  }
  return parse(body.data)
}

/**
 * The [from, to] calendar range of a month's billing period as wall-clock dates
 * (`YYYY-MM-DD`, both inclusive) — periodBoundsISO with this user's settings
 * filled in. See there for the running-vs-closed-period rules.
 *
 * These stay DATES, not instants: every caller sends them through
 * dayStartInstant/dayEndInstant itself. Handing instants out here once meant
 * fetchTransactions converted them a second time, which threw on an already-
 * converted string and left the dashboard with no rows at all.
 */
function periodRange(month: string): { from: string; to: string } {
  return periodBoundsISO(month, readToday(), readMonthStartDay())
}

/** Bounds of the billing period immediately BEFORE `month`'s — the base the
 *  summary compares against (per-category spend delta, carried-in leftover
 *  income). Same "which key does this month actually open in" step as
 *  periodRange, one period further back; always a *closed* period, so it runs
 *  start → end (never "→ today", unlike the current period). Same shift the
 *  transactions screen already uses for its "previous period" preset. */
function prevPeriodRange(month: string): { from: string; to: string } {
  const startDay = readMonthStartDay()
  const today = readToday()
  const openKey =
    month === currentMonthKey(today) ? periodOpenKey(today, startDay) : month
  const prevKey = shiftMonth(openKey, -1)

  return {
    from: periodStartISO(prevKey, startDay),
    to: periodEndISO(prevKey, startDay),
  }
}

export async function fetchCategories(): Promise<Category[]> {
  // GET /categories serves both dictionaries (expense + income), each tagged
  // with its `direction`; consumers filter on it (add sheet, categories screen).
  // Edits (PUT) and deletes are real now, so a refetch reflects them directly.
  return call("/categories", { method: "GET" }, (d) =>
    z.array(categorySchema).parse(d),
  )
}

/** Транзакции за месяц — live с бэка (protected /transactions), обе стороны
 *  (income + expense), каждая строка несёт `direction`. По умолчанию диапазон
 *  [начало месяца, сегодня] для текущего месяца; для прошлых — весь месяц. */
/**
 * Fetch transactions in an explicit [from, to] date range (both YYYY-MM-DD and
 * both inclusive — the backend shifts `to` by +1 day so the whole picked day
 * falls inside its half-open filter). Drives the Transactions screen's range
 * filter; a single day (from === to) is fine.
 */
export async function fetchTransactionsRange(
  from: string,
  to: string,
): Promise<Transaction[]> {
  const rows = await call(
    `/transactions?from=${dayStartInstant(from)}&to=${dayEndInstant(to)}`,
    { method: "GET" },
    (d) => z.array(transactionSchema).parse(d),
  )
  // Backend order isn't guaranteed, so sort by date+time descending client-side
  // to interleave income and expense chronologically for the list.
  return [...rows].sort((a, b) => {
    const ka = a.date + a.time
    const kb = b.date + b.time
    return ka < kb ? 1 : ka > kb ? -1 : 0
  })
}

/** Current-period convenience wrapper (dashboard / analytics use a month key). */
export function fetchTransactions(month: string): Promise<Transaction[]> {
  const { from, to } = periodRange(month)
  return fetchTransactionsRange(from, to)
}

/** Money on hand: the stated anchor plus every entry made since it. */
export function fetchBalance(): Promise<Balance> {
  return call("/balance", { method: "GET" }, (d) => balanceSchema.parse(d))
}

/**
 * State the balance ("right now I have X"). This re-anchors it: entries made
 * before this moment are taken as already reflected in the figure, so only what
 * follows moves it. Negative is allowed — an overdrawn card is a real state.
 */
export function setBalance(balanceMinor: number): Promise<Balance> {
  return call(
    "/balance",
    { method: "PUT", body: JSON.stringify({ balanceMinor }) },
    (d) => balanceSchema.parse(d),
  )
}

export function fetchBudgets(month: string): Promise<Budget[]> {
  return call(`/budgets?month=${month}`, { method: "GET" }, (d) =>
    z.array(budgetSchema).parse(d),
  )
}

/** Month summary — spent total, income total, unspent income carried in from
 *  the previous period and per-category spend breakdown (with its delta vs
 *  that same previous period) come live from the backend in one call (GET
 *  /transactions/summary). prevFrom/prevTo tell the backend which period to
 *  compare against; `byCategory` is sorted desc by spend client-side (the
 *  overview takes the top rows). */
export async function fetchMonthSummary(month: string): Promise<MonthSummary> {
  const { from, to } = periodRange(month)
  const prev = prevPeriodRange(month)

  const summary = await call(
    `/transactions/summary?from=${dayStartInstant(from)}&to=${dayEndInstant(to)}` +
      `&prevFrom=${dayStartInstant(prev.from)}&prevTo=${dayEndInstant(prev.to)}`,
    { method: "GET" },
    (d) =>
      z
        .object({
          spentMinor: z.number().int().min(0),
          incomeMinor: z.number().int().min(0),
          carriedInMinor: z.number().int().min(0),
          byCategory: z.array(
            z.object({
              categoryId: z.string(),
              spentMinor: z.number().int().min(0),
              deltaPct: z.number().nullable(),
            }),
          ),
        })
        .parse(d),
  )

  return {
    month,
    spentMinor: summary.spentMinor,
    incomeMinor: summary.incomeMinor,
    carriedInMinor: summary.carriedInMinor,
    byCategory: [...summary.byCategory].sort(
      (a, b) => b.spentMinor - a.spentMinor,
    ),
  }
}

/** Last six months of total spend, oldest first (analytics trend bars) — live
 *  from the backend. It buckets by the *billing* period (honouring the user's
 *  month-start-day setting), so a bar means the same span the dashboard shows,
 *  and months without spending come back as zero bars rather than gaps. */
export function fetchTrend(): Promise<TrendPoint[]> {
  // The backend groups instants into months; only we know which months those
  // are for this user, so the zone travels with the request.
  return call(
    `/transactions/trend?months=${trendMonths}&tz=${encodeURIComponent(browserTimezone())}`,
    { method: "GET" },
    (d) => z.array(trendPointSchema).parse(d),
  )
}

/** Rate sources for the dashboard's cycling pill — Br per one unit of the
 *  user's chosen currency (public /rates; NBRB + banks). Never called with
 *  BYN: that is the currency the amounts are in, so nothing is converted. */
export function fetchRates(currency: string): Promise<RateSource[]> {
  return call(`/rates?currency=${currency}`, { method: "GET" }, (d) =>
    z.array(rateSourceSchema).parse(d),
  )
}

/**
 * Static AI insights (the design's mock brain). Copy lives in messages under
 * `insights.<slug>.*`; numbers that the copy interpolates are here.
 */
/** AI insights — live from the backend (rules over the month's spending). Empty
 *  array means no advice this month; the UI shows a default message. */
export function fetchInsights(): Promise<Insights> {
  const month = currentMonthKey(readToday())
  const tz = encodeURIComponent(browserTimezone())

  return call(`/insights?month=${month}&tz=${tz}`, { method: "GET" }, (d) =>
    insightsSchema.parse(d),
  )
}

/**
 * Suggested limits per category — the "AI советует" pill in the limit sheet.
 * Live from the backend: the median of what the category actually cost over the
 * last three closed periods. Categories with too little history are simply
 * absent from the map, and the pill then doesn't render.
 */
export async function fetchLimitRecommendations(): Promise<
  Record<string, number>
> {
  const recs = await call(
    `/budgets/recommendations?tz=${encodeURIComponent(browserTimezone())}`,
    { method: "GET" },
    (d) =>
      z
        .array(
          z.object({
            categoryId: z.string(),
            limitMinor: z.number().int().positive(),
          }),
        )
        .parse(d),
  )

  return Object.fromEntries(recs.map((r) => [r.categoryId, r.limitMinor]))
}

export function createTransaction(
  input: AddTransactionInput,
): Promise<Transaction> {
  // Both directions persist on the real backend now: it carries a `direction`
  // column and validates that income uses an income category (else 400). Send
  // the full body including `direction`.
  const body = {
    amountMinor: input.amountMinor,
    categoryId: input.categoryId,
    title: input.title,
    // The sheet works in wall-clock; the wire is UTC.
    spentAt: toInstant(input.date, input.time),
    direction: input.direction,
  }
  return call(
    "/transactions",
    { method: "POST", body: JSON.stringify(body) },
    (d) => transactionSchema.parse(d),
  )
}

/** Edit an existing transaction (same fields/validation as create). */
export function updateTransaction(
  input: AddTransactionInput & { id: string },
): Promise<Transaction> {
  const body = {
    amountMinor: input.amountMinor,
    categoryId: input.categoryId,
    title: input.title,
    spentAt: toInstant(input.date, input.time),
    direction: input.direction,
  }
  return call(
    `/transactions/${input.id}`,
    { method: "PUT", body: JSON.stringify(body) },
    (d) => transactionSchema.parse(d),
  )
}

export function createCategory(input: CreateCategoryInput): Promise<Category> {
  // Both dictionaries persist on the real backend now (GET /categories serves
  // income + expense). `kind` becomes the `direction` on the row.
  return call(
    "/categories",
    {
      method: "POST",
      body: JSON.stringify({
        name: input.name,
        color: input.color,
        direction: input.kind,
      }),
    },
    (d) => categorySchema.parse(d),
  )
}

/** Rename/recolour a category. */
export function updateCategory(input: UpdateCategoryInput): Promise<Category> {
  // PUT updates name+color only — direction is immutable, so `kind` isn't sent.
  // Presets are locked in the UI, so only personal rows reach here.
  return call(
    `/categories/${input.id}`,
    {
      method: "PUT",
      body: JSON.stringify({ name: input.name, color: input.color }),
    },
    (d) => categorySchema.parse(d),
  )
}

/** Delete a personal category (presets are never deletable in the UI). */
export function deleteCategory(input: DeleteCategoryInput): Promise<void> {
  // Categories of either direction delete on the real backend: it removes the
  // caller's own personal row (presets / others' → 404). The envelope carries
  // `{ id }`; we ignore it and let the categories refetch reflect the removal.
  return call<void>(
    `/categories/${input.id}`,
    { method: "DELETE" },
    () => undefined,
  )
}

/** Delete one of the caller's own transactions (others' / missing → 404). */
export function deleteTransaction(id: string): Promise<void> {
  // Mirrors deleteCategory: the envelope carries `{ id }`; we ignore it and let
  // the ["budget"] refetch reflect the removal.
  return call<void>(
    `/transactions/${id}`,
    { method: "DELETE" },
    () => undefined,
  )
}

/**
 * Set a category's limit from `month` onward — and every month after it, until
 * it is changed again. Re-saving in the same month corrects that run rather than
 * opening a new one.
 */
export function saveBudget(input: SaveBudgetInput): Promise<Budget> {
  return call(
    "/budgets",
    { method: "POST", body: JSON.stringify(input) },
    (d) => budgetSchema.parse(d),
  )
}

/**
 * Remove a category's limit from `month` onward. Earlier months keep theirs —
 * dropping a limit today says nothing about what it was in March.
 */
export function deleteBudget(categoryId: string, month: string): Promise<void> {
  return call<void>(
    `/budgets/${categoryId}?month=${month}`,
    { method: "DELETE" },
    () => undefined,
  )
}

export function fetchPlans(month: string): Promise<Plan[]> {
  return call(`/plans?month=${month}`, { method: "GET" }, (d) =>
    z.array(planSchema).parse(d),
  )
}

/** Plans the backend suggests for the month, from what was confirmed before. */
export function fetchPlanSuggestions(month: string): Promise<PlanSuggestion[]> {
  return call(`/plans/suggestions?month=${month}`, { method: "GET" }, (d) =>
    z.array(planSuggestionSchema).parse(d),
  )
}

export function createPlan(input: {
  month: string
  categoryId: string
  title: string
  amountMinor: number
}): Promise<Plan> {
  return call("/plans", { method: "POST", body: JSON.stringify(input) }, (d) =>
    planSchema.parse(d),
  )
}

export function updatePlan(input: {
  id: string
  categoryId: string
  title: string
  amountMinor: number
}): Promise<Plan> {
  const { id, ...body } = input
  return call(
    `/plans/${id}`,
    { method: "PUT", body: JSON.stringify(body) },
    (d) => planSchema.parse(d),
  )
}

/** Mark a plan confirmed; the transaction is created separately via the add flow. */
export function confirmPlan(id: string): Promise<Plan> {
  return call(`/plans/${id}/confirm`, { method: "POST" }, (d) =>
    planSchema.parse(d),
  )
}

export function deletePlan(id: string): Promise<void> {
  return call<void>(`/plans/${id}`, { method: "DELETE" }, () => undefined)
}
