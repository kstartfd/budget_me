"use client"

import { useRouter, useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"

import {
  useConfirmDayEmail,
  useDayEmailPreference,
  useUpdateDayEmailPreference,
} from "./hooks"

export function DayEmailSection() {
  const t = useTranslations("dayEmail")
  const preference = useDayEmailPreference()
  const update = useUpdateDayEmailPreference()
  const confirm = useConfirmDayEmail()
  const router = useRouter()
  const token = useSearchParams().get("dayEmailToken")

  function finishConfirmation() {
    if (!token) return
    confirm.mutate(token, {
      onSuccess: () => {
        router.replace("/settings")
      },
    })
  }

  const data = preference.data
  const busy = update.isPending || confirm.isPending

  return (
    <section className="flex flex-col gap-[9px]">
      <span className="text-faint px-0.5 font-mono text-[10px] font-semibold tracking-[0.07em] uppercase">
        {t("kicker")}
      </span>
      <div className="bg-card border-border rounded-[18px] border p-[15px] shadow-[var(--shadow-card)]">
        <div className="text-[13.5px] font-semibold">{t("title")}</div>
        <p className="text-muted-foreground mt-1 text-xs">{t("hint")}</p>

        {token && (
          <div className="bg-secondary mt-3 rounded-xl p-3">
            <p className="text-sm font-medium">{t("confirmHint")}</p>
            <button
              type="button"
              onClick={finishConfirmation}
              disabled={confirm.isPending}
              className="bg-primary text-primary-foreground mt-2 rounded-xl px-4 py-2 text-sm font-semibold disabled:opacity-40"
            >
              {t("confirm")}
            </button>
            {confirm.isError && (
              <p role="alert" className="text-destructive mt-2 text-xs">
                {t("confirmError")}
              </p>
            )}
          </div>
        )}

        {preference.isPending && (
          <p className="text-muted-foreground mt-3 text-xs">{t("loading")}</p>
        )}
        {preference.isError && (
          <p role="alert" className="text-destructive mt-3 text-xs">
            {t("loadError")}
          </p>
        )}
        {data && (
          <>
            {!data.email ? (
              <p className="text-muted-foreground mt-3 text-xs">
                {t("linkEmail")}
              </p>
            ) : (
              <>
                <p className="text-faint mt-3 text-xs">{data.email}</p>
                {!data.configured && (
                  <p className="text-muted-foreground mt-2 text-xs">
                    {t("unavailable")}
                  </p>
                )}
                <div
                  role="group"
                  aria-label={t("title")}
                  className="mt-3 flex flex-wrap gap-2"
                >
                  {([0, 1, 2] as const).map((frequency) => (
                    <button
                      key={frequency}
                      type="button"
                      aria-pressed={data.frequency === frequency}
                      onClick={() => update.mutate(frequency)}
                      disabled={busy || (!data.configured && frequency !== 0)}
                      className={`rounded-xl px-3 py-2 text-xs font-semibold disabled:opacity-40 ${data.frequency === frequency ? "bg-primary text-primary-foreground" : "bg-secondary text-foreground"}`}
                    >
                      {t(`frequency.${frequency}`)}
                    </button>
                  ))}
                </div>
                {data.status === "pending" && (
                  <p className="text-muted-foreground mt-3 text-xs">
                    {t("pending")}{" "}
                    <button
                      type="button"
                      onClick={() => update.mutate(data.frequency as 1 | 2)}
                      disabled={busy || !data.configured}
                      className="text-primary font-semibold underline disabled:opacity-40"
                    >
                      {t("resend")}
                    </button>
                  </p>
                )}
                {data.status === "active" && (
                  <p className="text-muted-foreground mt-3 text-xs">
                    {t("active")}
                  </p>
                )}
                {update.isError && (
                  <p role="alert" className="text-destructive mt-2 text-xs">
                    {t("saveError")}
                  </p>
                )}
              </>
            )}
          </>
        )}
      </div>
    </section>
  )
}
