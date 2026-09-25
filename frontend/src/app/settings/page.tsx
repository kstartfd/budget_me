"use client"

import { Suspense, useState } from "react"
import Link from "next/link"
import { useQueryClient } from "@tanstack/react-query"
import { useFormatter, useLocale, useTranslations } from "next-intl"

import {
  ChevronRightIcon,
  ClockIcon,
  GridIcon,
  TargetIcon,
} from "@/components/app-icons"
import { Button } from "@/components/ui/button"
import { AccountSection, DeleteAccountSection } from "@/features/account"
import { useAuth } from "@/features/auth"
import { DayEmailSection } from "@/features/day-email/day-email-section"
import { budgetKeys } from "@/features/budget"
import { useBudgetUI } from "@/features/budget/components/ui-context"
import { CurrencySelect } from "@/features/settings/components/currency-select"
import { DayGrid } from "@/features/settings/components/day-grid"
import { DayWheel } from "@/features/settings/components/day-wheel"
import {
  isHistoryResetRequired,
  useSettings,
  useUpdateSettings,
} from "@/features/settings"
import type { Currency } from "@/features/settings"
import { periodLabel } from "@/lib/month"
import { SettingsSkeleton } from "./skeleton"

/**
 * Settings screen (design PHONE 4 / WEB FRAME 2 "settings side"): the billing
 * period start day as a wheel, the main currency, and a link toward the future
 * categories screen. Runs on the settings mock layer behind useSettings /
 * useUpdateSettings, the same hooks a real GET/PUT /settings will back.
 */
export default function SettingsPage() {
  const t = useTranslations("settings")
  const tCommon = useTranslations("common")
  const locale = useLocale()
  const format = useFormatter()
  const queryClient = useQueryClient()
  const { user } = useAuth()
  const { showToast } = useBudgetUI()

  const settings = useSettings()
  const update = useUpdateSettings()
  const [draftDay, setDraftDay] = useState<number | null>(null)
  // Set once the server has refused the change pending confirmation, which is
  // also the only thing that can raise it — the screen never guesses that a
  // reset is due, it waits to be told.
  const [confirmingReset, setConfirmingReset] = useState(false)

  if (settings.isPending) return <SettingsSkeleton />
  if (settings.isError) {
    return (
      <p
        role="alert"
        className="text-muted-foreground py-10 text-center text-sm"
      >
        {tCommon("loadError")}
      </p>
    )
  }

  const { monthStartDay, currency, today, historyResetAt } = settings.data
  // The picker edits a draft; the button below commits it. Defaults to the
  // saved day (1 for a fresh account), so the wheel opens on the 1st.
  const day = draftDay ?? monthStartDay
  const dirty = draftDay !== null && draftDay !== monthStartDay

  // Moving the day re-cuts every period the app has shown, so the server refuses
  // it (409) until the user agrees the running totals restart. The first attempt
  // therefore raises the confirmation rather than saving; the second carries it.
  function commitDay(resetHistory = false) {
    if (!dirty) return

    update.mutate(
      { monthStartDay: day, ...(resetHistory ? { resetHistory } : {}) },
      {
        // Shifting the period changes every budget read's date range — and a
        // reset also drops the balance anchor, which lives under the same key.
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: budgetKeys.all })
          setDraftDay(null)
          setConfirmingReset(false)

          if (resetHistory) showToast("info", t("resetDoneToast"))
        },
        onError: (error) => {
          if (isHistoryResetRequired(error)) {
            setConfirmingReset(true)
            return
          }

          showToast("danger", tCommon("loadError"))
        },
      },
    )
  }

  function cancelReset() {
    setConfirmingReset(false)
    setDraftDay(null)
  }

  function setCurrency(next: Currency) {
    update.mutate({ currency: next })
  }

  const userId = user?.id ?? ""

  function copyId() {
    if (!userId) return

    void navigator.clipboard?.writeText(userId)
    showToast("success", t("idCopied"))
  }

  return (
    <div className="anim-fade flex flex-col gap-[18px]">
      <h1 className="px-0.5 text-[22px] font-bold tracking-[-0.02em]">
        {t("title")}
      </h1>

      {/* Your ID — the account uuid, copyable (support / account ops) */}
      <section className="flex flex-col gap-[9px]">
        <div className="bg-card border-border flex items-center gap-3 rounded-[18px] border p-[14px] shadow-[var(--shadow-card)]">
          <div className="min-w-0 flex-1">
            <div className="text-[13px] font-medium">{t("yourId")}</div>
            <div className="text-faint mt-[3px] truncate font-mono text-[11.5px]">
              {userId}
            </div>
          </div>
          <button
            type="button"
            onClick={copyId}
            aria-label={t("copyId")}
            className="border-border bg-secondary text-muted-foreground flex size-[34px] shrink-0 items-center justify-center rounded-full border"
          >
            <svg
              width="15"
              height="15"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden
            >
              <rect x="9" y="9" width="13" height="13" rx="2" />
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
            </svg>
          </button>
        </div>
      </section>

      {/* Billing period — month start wheel */}
      <section className="flex flex-col gap-[9px]">
        <span className="text-faint px-0.5 font-mono text-[10px] font-semibold tracking-[0.07em] uppercase">
          {t("periodKicker")}
        </span>
        <div className="bg-card border-border rounded-[18px] border px-[15px] pt-[15px] pb-[13px] shadow-[var(--shadow-card)]">
          <div className="text-[13.5px] font-semibold">{t("monthStart")}</div>
          <p className="text-muted-foreground mt-[3px] text-xs font-medium">
            {t("monthStartHint")}
          </p>

          {/* Wheel on touch/phone; a calendar-style day grid reads clearer on
              wide pointer screens (design "2b calendar"). Same value + setter —
              only the presentation switches by breakpoint. */}
          <div className="mt-3 lg:hidden">
            <DayWheel
              value={day}
              onChange={setDraftDay}
              label={t("wheelLabel")}
            />
          </div>
          <div className="mt-3 hidden lg:block">
            <DayGrid
              value={day}
              onChange={setDraftDay}
              label={t("wheelLabel")}
            />
          </div>

          <p className="text-faint mt-2 text-center text-[11px] font-medium">
            {t("clampHint")}
          </p>

          <div className="bg-[color:var(--primary-wash)] mt-3 flex items-center gap-2 rounded-xl px-3 py-[9px]">
            <span className="text-primary shrink-0">
              <ClockIcon />
            </span>
            <span className="text-[12px] font-semibold">
              {t("currentPeriod")}{" "}
              <span className="text-primary">
                {periodLabel(today, day, locale)}
              </span>
            </span>
          </div>

          {confirmingReset ? (
            <div className="border-[color:var(--destructive-line,var(--border))] mt-3 rounded-xl border p-3">
              <div className="text-destructive text-[13px] font-semibold">
                {t("resetTitle", { day })}
              </div>
              <p className="text-muted-foreground mt-[3px] text-[11.5px]">
                {t("resetHint")}
              </p>
              <div className="mt-3 flex gap-2">
                <button
                  type="button"
                  onClick={cancelReset}
                  className="border-border bg-secondary text-foreground flex-1 rounded-xl p-2.5 text-[13px] font-semibold"
                >
                  {t("resetCancel")}
                </button>
                <button
                  type="button"
                  onClick={() => commitDay(true)}
                  disabled={update.isPending}
                  className="bg-destructive text-destructive-foreground flex-1 rounded-xl p-2.5 text-[13px] font-bold disabled:opacity-60"
                >
                  {update.isPending ? t("setStartSaving") : t("resetConfirm")}
                </button>
              </div>
            </div>
          ) : (
            <Button
              onClick={() => commitDay()}
              disabled={!dirty || update.isPending}
              className="mt-3 h-10 w-full rounded-xl"
            >
              {update.isPending ? t("setStartSaving") : t("setStart")}
            </Button>
          )}

          {historyResetAt ? (
            <p className="text-faint mt-2.5 text-center text-[11px] font-medium">
              {t("resetSince", {
                date: format.dateTime(new Date(historyResetAt), {
                  day: "numeric",
                  month: "long",
                }),
              })}
            </p>
          ) : null}
        </div>
      </section>

      {/* Main currency */}
      <section className="flex flex-col gap-[9px]">
        <span className="text-faint px-0.5 font-mono text-[10px] font-semibold tracking-[0.07em] uppercase">
          {t("currencyKicker")}
        </span>
        <div className="bg-card border-border rounded-[18px] border p-[14px] shadow-[var(--shadow-card)]">
          <div className="flex items-center gap-3">
            <div className="flex-1">
              <div className="text-[13px] font-medium">{t("mainCurrency")}</div>
              <p className="text-muted-foreground mt-[3px] text-[11.5px]">
                {t("currencyHint")}
              </p>
            </div>
            <CurrencySelect
              value={currency}
              onChange={setCurrency}
              label={t("mainCurrency")}
            />
          </div>
        </div>
      </section>

      {/* Categories + Limits — links to their own screens */}
      <section className="flex flex-col gap-[9px]">
        <span className="text-faint px-0.5 font-mono text-[10px] font-semibold tracking-[0.07em] uppercase">
          {t("moreKicker")}
        </span>
        <Link
          href="/categories"
          className="bg-card border-border flex items-center gap-3 rounded-[18px] border p-[14px] shadow-[var(--shadow-card)]"
        >
          <span className="bg-[color:var(--primary-wash)] text-primary flex size-[38px] shrink-0 items-center justify-center rounded-xl">
            <GridIcon />
          </span>
          <div className="min-w-0 flex-1">
            <div className="text-[14px] font-semibold">{t("categories")}</div>
            <div className="text-faint mt-[2px] text-[11.5px]">
              {t("categoriesSub")}
            </div>
          </div>
          <span className="text-faint shrink-0">
            <ChevronRightIcon />
          </span>
        </Link>
        <Link
          href="/budgets"
          className="bg-card border-border flex items-center gap-3 rounded-[18px] border p-[14px] shadow-[var(--shadow-card)]"
        >
          <span className="bg-[color:var(--primary-wash)] text-primary flex size-[38px] shrink-0 items-center justify-center rounded-xl">
            <TargetIcon />
          </span>
          <div className="min-w-0 flex-1">
            <div className="text-[14px] font-semibold">{t("limits")}</div>
            <div className="text-faint mt-[2px] text-[11.5px]">
              {t("limitsSub")}
            </div>
          </div>
          <span className="text-faint shrink-0">
            <ChevronRightIcon />
          </span>
        </Link>
        <Link
          href="/plans"
          className="bg-card border-border flex items-center gap-3 rounded-[18px] border p-[14px] shadow-[var(--shadow-card)]"
        >
          <span className="bg-[color:var(--primary-wash)] text-primary flex size-[38px] shrink-0 items-center justify-center rounded-xl">
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden
            >
              <path d="M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2" />
              <rect x="9" y="3" width="6" height="4" rx="1" />
              <path d="M9 13l2 2 4-4" />
            </svg>
          </span>
          <div className="min-w-0 flex-1">
            <div className="text-[14px] font-semibold">{t("plans")}</div>
            <div className="text-faint mt-[2px] text-[11.5px]">
              {t("plansSub")}
            </div>
          </div>
          <span className="text-faint shrink-0">
            <ChevronRightIcon />
          </span>
        </Link>
      </section>

      {/* Account — link Telegram ↔ email so one account has both login paths */}
      <AccountSection />

      <Suspense fallback={null}>
        <DayEmailSection />
      </Suspense>

      {/* Danger zone — deferred account deletion (30-day grace, cancellable) */}
      <DeleteAccountSection />
    </div>
  )
}
