"use client"

import Link from "next/link"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { card, Pill } from "@/features/home/components/parts"
import type { AssessmentAction } from "@/features/path/api"
import { cn } from "@/lib/utils"

import { AssessmentHeading } from "./start"

export function AssessmentLoading({ title }: { title: string }) {
  return (
    <section
      aria-busy="true"
      role="status"
      className={cn(card, "max-w-[720px] space-y-4 p-5")}
    >
      <p className="font-semibold">{title}</p>
      <Skeleton className="h-4 w-4/5" />
      <Skeleton className="h-4 w-3/5" />
      <Skeleton className="h-10" />
    </section>
  )
}

export function AssessmentRecovery({
  title,
  error,
  onRetry,
  onChoose,
}: {
  title: string
  error: string
  onRetry: () => void
  onChoose?: () => void
}) {
  const t = useTranslations("assessment")

  return (
    <section className={cn(card, "max-w-[720px] space-y-3 p-5")}>
      <h1 className="text-lg font-semibold">{title}</h1>
      <p role="alert" className="text-muted-foreground text-sm">
        {error}
      </p>
      <div className="flex flex-wrap gap-2">
        <Button className="h-11 px-4 text-sm" onClick={onRetry}>
          {t("retry")}
        </Button>
        {onChoose && (
          <Button
            variant="outline"
            className="h-11 px-4 text-sm"
            onClick={onChoose}
          >
            {t("otherStart")}
          </Button>
        )}
      </div>
    </section>
  )
}

export function AssessmentResume({
  language,
  answered,
  paused,
  pending,
  error,
  onAction,
}: {
  language: string
  answered: number
  paused: boolean
  pending: boolean
  error?: string
  onAction: (action: AssessmentAction) => void
}) {
  const t = useTranslations("assessment")

  return (
    <div className="space-y-5">
      <AssessmentHeading title={t("title")} />
      <section className={cn(card, "max-w-[760px] space-y-4 p-5 sm:p-6")}>
        <Pill tone="learning">{t(paused ? "paused" : "unfinished")}</Pill>
        <div className="space-y-1">
          <h2 className="text-xl font-semibold">
            {t("resumeTitle", { language })}
          </h2>
          <p className="text-muted-foreground text-sm">{t("resumeText")}</p>
        </div>
        <dl className="bg-muted grid gap-3 rounded-lg p-4 text-sm sm:grid-cols-3">
          <div>
            <dt className="text-muted-foreground text-xs">{t("language")}</dt>
            <dd>{language}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-xs">{t("type")}</dt>
            <dd>{t("option.quick.title")}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-xs">{t("answered")}</dt>
            <dd>{t("answerCount", { count: answered })}</dd>
          </div>
        </dl>
        {error && (
          <p role="alert" className="text-destructive text-sm">
            {error}
          </p>
        )}
        <div className="flex flex-wrap items-center gap-2">
          <Button
            className="h-11 rounded-lg px-4 text-sm"
            disabled={pending}
            onClick={() => onAction("resume")}
          >
            {t("resume")}
          </Button>
          {answered > 0 && (
            <Button
              variant="outline"
              className="h-11 rounded-lg px-4 text-sm"
              disabled={pending}
              onClick={() => onAction("complete")}
            >
              {t("finishEarly")}
            </Button>
          )}
          <Dialog>
            <DialogTrigger asChild>
              <Button
                variant="outline"
                className="h-11 rounded-lg px-4 text-sm"
                disabled={pending}
              >
                {t("restart")}
              </Button>
            </DialogTrigger>
            <DialogContent
              showCloseButton={false}
              onEscapeKeyDown={(event) => {
                if (pending) event.preventDefault()
              }}
              onInteractOutside={(event) => {
                if (pending) event.preventDefault()
              }}
            >
              <DialogHeader>
                <DialogTitle>{t("restartTitle")}</DialogTitle>
                <DialogDescription>
                  {t("restartText", { count: answered })}
                </DialogDescription>
              </DialogHeader>
              {error && (
                <p role="alert" className="text-destructive text-sm">
                  {error}
                </p>
              )}
              <DialogFooter>
                <DialogClose asChild>
                  <Button variant="outline" className="h-11" disabled={pending}>
                    {t("keepAnswers")}
                  </Button>
                </DialogClose>
                <Button
                  variant="destructive"
                  className="h-11"
                  disabled={pending}
                  onClick={() => onAction("cancel")}
                >
                  {t("restartConfirm")}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
          <Button asChild variant="link" className="h-11 text-sm">
            <Link href="/learning/overview">{t("overview")}</Link>
          </Button>
        </div>
      </section>
    </div>
  )
}
