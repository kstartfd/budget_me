import { z } from "zod"

import { fromInstant } from "@/lib/time"

/**
 * Domain schemas for the budget feature ("Budget Mini App" design). The app
 * tracks both expenses and income in one currency (Br). These mirror the
 * payload shapes the Go backend will eventually serve; while the app runs on
 * the mock data layer they are also the storage shape, so swapping the
 * fetchers later doesn't touch components.
 *
 * Money is integer minor units (kopecks of Br) everywhere; formatting
 * ("142,80 Br", "≈ $44") happens only at render time (see lib/money.ts).
 */

/** A month key in `YYYY-MM` form — the app's unit of budgeting. */
export const monthKeySchema = z.string().regex(/^\d{4}-\d{2}$/)
export type MonthKey = z.infer<typeof monthKeySchema>

/**
 * Income vs expense. `amountMinor` always stays a positive magnitude — the
 * direction carries the sign meaning, not a negative amount.
 */
export const directionSchema = z.enum(["income", "expense"])
export type Direction = z.infer<typeof directionSchema>

export const categorySchema = z.object({
  id: z.string(),
  /** i18n slug for the default categories; copy comes from `categories.<slug>`. */
  slug: z.string().optional(),
  /** Free name for user-added categories — shown as-is, no i18n. */
  name: z.string().optional(),
  /** Category hue (hex) — fixed per category on every screen. */
  color: z.string().regex(/^#[0-9a-f]{6}$/i),
  /**
   * Income or expense — the backend serves both dictionaries from GET
   * /categories, each tagged with its direction. The add sheet / categories
   * screen filter on it. Defaults to expense so mock-seeded rows (and any older
   * row without the column) still parse.
   */
  direction: directionSchema.default("expense"),
})
export type Category = z.infer<typeof categorySchema>

/** A hex colour like `#7c86ff` — the category hue on every screen. */
const hexColor = z.string().regex(/^#[0-9a-f]{6}$/i)

/**
 * Input for creating a category (name + hue). `kind` is the category direction
 * (income/expense) — it becomes the `direction` on the `POST /categories` body.
 */
export const createCategoryInputSchema = z.object({
  kind: directionSchema,
  name: z.string().trim().min(1),
  color: hexColor,
})
export type CreateCategoryInput = z.infer<typeof createCategoryInputSchema>

/** Input for renaming/recolouring a category. Real: `PUT /categories/{id}`
 *  with `{ name, color }` (`kind` tags which tab it came from). */
export const updateCategoryInputSchema = z.object({
  id: z.string(),
  kind: directionSchema,
  name: z.string().trim().min(1),
  color: hexColor,
})
export type UpdateCategoryInput = z.infer<typeof updateCategoryInputSchema>

/** Input for deleting a category. Real: `DELETE /categories/{id}`. */
export const deleteCategoryInputSchema = z.object({
  id: z.string(),
  kind: directionSchema,
})
export type DeleteCategoryInput = z.infer<typeof deleteCategoryInputSchema>

/**
 * A transaction as the backend serves it: the moment of the spend is a UTC
 * instant (`spentAt`), because only the client knows which day that is for the
 * user. The parse splits it into the local `date`/`time` the whole UI renders
 * from, so screens keep working with wall-clock and never touch zones.
 */
export const transactionSchema = z
  .object({
    id: z.string(),
    /** Positive magnitude in minor units; `direction` carries the sign meaning. */
    amountMinor: z.number().int().positive(),
    categoryId: z.string(),
    /** Merchant / free-form title shown as the row title. */
    title: z.string(),
    /** The instant of the spend, RFC3339 UTC. */
    spentAt: z.string(),
    direction: directionSchema.default("expense"),
  })
  .transform((t) => ({ ...t, ...fromInstant(t.spentAt) }))
export type Transaction = z.infer<typeof transactionSchema>

export const budgetSchema = z.object({
  categoryId: z.string(),
  month: monthKeySchema,
  limitMinor: z.number().int().positive(),
})
export type Budget = z.infer<typeof budgetSchema>

/** A planned expense for a month — confirmed once the spend actually happens. */
export const planSchema = z.object({
  id: z.string(),
  month: monthKeySchema,
  categoryId: z.string(),
  title: z.string(),
  amountMinor: z.number().int().positive(),
  confirmed: z.boolean(),
})
export type Plan = z.infer<typeof planSchema>

/**
 * A plan the backend proposes for the month, rebuilt from what the user
 * confirmed in the last few months. `seenCount` is how many of those months
 * carried it — the UI shows it as "3 месяца подряд".
 */
export const planSuggestionSchema = z.object({
  categoryId: z.string(),
  title: z.string(),
  amountMinor: z.number().int().positive(),
  seenCount: z.number().int().positive(),
})
export type PlanSuggestion = z.infer<typeof planSuggestionSchema>

/**
 * Money on hand. `set` is false until the user has stated an opening figure —
 * transactions alone cannot know what was already in the wallet, so the app
 * asks instead of showing a zero it invented.
 */
export const balanceSchema = z.object({
  set: z.boolean(),
  balanceMinor: z.number().int(),
})
export type Balance = z.infer<typeof balanceSchema>

/** Aggregated month view the dashboard/analytics screens render from. */
export const monthSummarySchema = z.object({
  month: monthKeySchema,
  spentMinor: z.number().int().min(0),
  /** Total income (приход) logged in the period — «Осталось» = incomeMinor − spentMinor. */
  incomeMinor: z.number().int().min(0),
  /** Unspent income carried in from the immediately preceding period (0 if that
   *  period was fully spent, or overspent — never a debt). Shown as its own line,
   *  never folded into incomeMinor: incomeMinor stays "money that arrived this
   *  period" per the money-info-sheet explainer. */
  carriedInMinor: z.number().int().min(0),
  byCategory: z.array(
    z.object({
      categoryId: z.string(),
      spentMinor: z.number().int().min(0),
      /** Percent change vs the previous month, null when no base. */
      deltaPct: z.number().nullable(),
    }),
  ),
})
export type MonthSummary = z.infer<typeof monthSummarySchema>

/** One bar of the 6-month trend chart. */
export const trendPointSchema = z.object({
  month: monthKeySchema,
  spentMinor: z.number().int().min(0),
})
export type TrendPoint = z.infer<typeof trendPointSchema>

/** A currency-rate source for the ≈USD conversion pill. */
export const rateSourceSchema = z.object({
  /** Slug for messages under `rates.<slug>`. */
  slug: z.string(),
  /** Br per 1 USD. */
  rate: z.number().positive(),
})
export type RateSource = z.infer<typeof rateSourceSchema>

/**
 * One card on the AI screen. Every figure comes FROM THE BACKEND — the previous
 * version kept its numbers inside the translation strings, where they were
 * simply made up ("Netflix and Spotify unused for 3 weeks").
 *
 * `overspend` — this category broke its limit; `noLimit` — the biggest category
 * has no limit and here is one to try; `onTrack` — limits exist and all hold.
 */
export const insightSchema = z.object({
  kind: z.enum(["overspend", "noLimit", "onTrack"]),
  tone: z.enum(["info", "warn", "success", "danger"]),
  categoryId: z.string().optional(),
  spentMinor: z.number().int().optional(),
  limitMinor: z.number().int().optional(),
  suggestMinor: z.number().int().optional(),
})
export type Insight = z.infer<typeof insightSchema>

/**
 * What the AI screen renders. `overspendMinor` is computed by the backend from
 * the user's own limits (it replaced a number that used to be hardcoded on this
 * screen). `advice` is the model's wording of those same facts — null until it
 * has been written: the model needs seconds, so it never blocks a request and
 * the text simply appears on a later open.
 */
export const insightsSchema = z.object({
  insights: z.array(insightSchema),
  overspendMinor: z.number().int().min(0),
  advice: z.string().nullable(),
})
export type Insights = z.infer<typeof insightsSchema>

/**
 * What the add/edit sheet produces: wall-clock date+time as the user picked
 * them. The api layer turns that into the UTC instant the backend wants.
 */
export const addTransactionInputSchema = z.object({
  amountMinor: z.number().int().positive(),
  categoryId: z.string(),
  title: z.string(),
  /** Local calendar date `YYYY-MM-DD` as shown to the user. */
  date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
  /** Local wall-clock `HH:MM`. */
  time: z.string().regex(/^\d{2}:\d{2}$/),
  direction: directionSchema.default("expense"),
})
export type AddTransactionInput = z.infer<typeof addTransactionInputSchema>
