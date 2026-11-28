"use client";

import { useState } from "react";
import Link from "next/link";
import { useLocale, useTranslations } from "next-intl";
import {
  IconCalendar,
  IconCheck,
  IconClock,
  IconPlayerPlay,
  IconRefresh,
  IconSparkles,
} from "@tabler/icons-react";

import { Button } from "@/components/ui/button";
import {
  useLesson,
  useLessonTotals,
  useLoopDashboard,
} from "@/features/path/hooks";
import type {
  LoopActivity,
  Path,
  PathLesson,
  PathTeacher,
} from "@/features/path/schemas";
import { cn } from "@/lib/utils";

import { skillIds } from "../profile";
import { Caption, Pill, card, type Tone } from "./parts";
import { useLanguageName } from "./language-switcher";
import { ReviewActivity } from "./review-activity";

/** One lesson is thirty minutes: the server closes it at that mark. */
const lessonMinutes = 30;
const dayMs = 24 * 60 * 60 * 1000;

/** "2h 15m", "45m", "0m": time studied, from seconds. */
export function formatStudy(seconds: number): string {
  const minutes = Math.round(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours === 0) return `${minutes}m`;

  return minutes % 60 === 0 ? `${hours}h` : `${hours}h ${minutes % 60}m`;
}

/** The seconds a lesson was studied for: its time, less the pauses. */
function studiedSeconds(lesson: PathLesson, now: number): number {
  const start = Date.parse(lesson.startedAt);
  const end = lesson.finishedAt
    ? Date.parse(lesson.finishedAt)
    : Math.min(now, Date.parse(lesson.endsAt));

  return Math.max(0, (end - start) / 1000 - lesson.pausedSeconds);
}

type Bar = {
  key: string;
  label: string;
  minutes: number;
  today: boolean;
  future: boolean;
};

/** Monday to Sunday of the week `now` is in, with the minutes studied each day. */
function weekBars(lessons: PathLesson[], now: number, locale: string): Bar[] {
  const today = new Date(now);
  const monday = new Date(
    today.getFullYear(),
    today.getMonth(),
    today.getDate() - ((today.getDay() + 6) % 7),
  );
  const names = new Intl.DateTimeFormat(locale, { weekday: "short" });

  return Array.from({ length: 7 }, (_, index) => {
    const day = new Date(
      monday.getFullYear(),
      monday.getMonth(),
      monday.getDate() + index,
    );
    const next = new Date(
      monday.getFullYear(),
      monday.getMonth(),
      monday.getDate() + index + 1,
    );
    const seconds = lessons
      .filter((lesson) => {
        const started = Date.parse(lesson.startedAt);

        return started >= day.getTime() && started < next.getTime();
      })
      .reduce((sum, lesson) => sum + studiedSeconds(lesson, now), 0);

    return {
      key: day.toISOString(),
      label: names.format(day),
      minutes: Math.round(seconds / 60),
      today: day.getTime() <= now && now < next.getTime(),
      future: day.getTime() > now,
    };
  });
}

/**
 * The overview of a language with a running program: what to do next, how the
 * week is going, what is due. Everything is read from the program, the lessons
 * and the learning loop; what is not measured yet says so instead of showing a
 * number.
 */
export function Dashboard({
  teacher,
  path,
  aiReady,
}: {
  teacher: PathTeacher;
  path: Path;
  aiReady: boolean;
}) {
  const t = useTranslations("home");
  const locale = useLocale();
  const name = useLanguageName();
  const loop = useLoopDashboard(teacher.language);
  const totals = useLessonTotals();
  const { history } = useLesson();
  // The clock of this screen: when the lessons were last read, or when it opened.
  const [openedAt] = useState(() => Date.now());
  const [selectedReview, setSelectedReview] = useState<LoopActivity | null>(
    null,
  );
  const now = history.dataUpdatedAt || totals.dataUpdatedAt || openedAt;

  const topics = [...path.topics].sort((a, b) => a.position - b.position);
  const done = topics.filter((topic) => topic.status === "completed").length;
  const current = topics.find((topic) => topic.status !== "completed") ?? null;
  const upcoming =
    topics.filter((topic) => topic.status !== "completed")[1] ?? null;
  const finished = topics.length > 0 && !current;
  const programHref = `/learning/path?language=${teacher.language}`;
  const continueHref = current
    ? `${programHref}&topic=${encodeURIComponent(current.id)}`
    : programHref;

  const reviews = loop.data?.reviews;
  const items = loop.data?.items;
  const reviewActivities = (loop.data?.activities ?? []).filter(
    (activity) => activity.type === "review" && activity.exercise,
  );
  const practice =
    (loop.data?.activities ?? []).find(
      (activity) => activity.type === "practice",
    ) ?? null;

  const bars = weekBars(history.data ?? [], now, locale);
  const activeDays = bars.filter((bar) => bar.minutes > 0).length;
  const weekSeconds = totals.data?.week ?? 0;
  const goalSeconds = path.weeklyGoalMinutes * 60;
  const dueCount = reviews?.due ?? 0;

  const plan: PlanRow[] = [];
  if (dueCount > 0)
    plan.push({
      key: "review",
      type: t("dash.typeReview"),
      title: t("dash.reviewRow", { count: dueCount }),
      objective: t("dash.reviewObjective"),
      status: t("dash.statusDue"),
      tone: "review",
      href: null,
    });
  if (practice && current)
    plan.push({
      key: practice.id,
      type: skillIds.some((skill) => skill === practice.item?.skill)
        ? t(`skill.${practice.item?.skill as (typeof skillIds)[number]}`)
        : t("dash.typePractice"),
      title: current.title,
      objective: current.goal,
      status: t(
        current.status === "in_progress"
          ? "dash.statusInProgress"
          : "dash.statusNotStarted",
      ),
      tone: current.status === "in_progress" ? "learning" : "muted",
      href: continueHref,
    });

  const why =
    dueCount > 0
      ? t("dash.whyDue", { count: dueCount })
      : current
        ? t("dash.whyNext", { title: current.title })
        : t("dash.whyDone");

  return (
    <div className="space-y-6">
      {selectedReview && (
        <ReviewActivity
          key={selectedReview.id}
          activity={selectedReview}
          onClose={() => setSelectedReview(null)}
        />
      )}
      {!aiReady && <AiNotice />}
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,420px),1fr))] items-start gap-6">
        <div className="flex min-w-0 flex-col gap-6">
          <section
            aria-labelledby="continue"
            className={cn(card, "flex flex-col gap-4 p-6")}
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <Caption>
                {finished ? t("dash.planCompleted") : t("dash.continue")}
              </Caption>
              <Pill tone="learning">
                <IconClock size={12} aria-hidden="true" />
                {t("dash.pathBadge", { level: path.level })}
              </Pill>
            </div>
            <div>
              <h2
                id="continue"
                className="text-[22px] font-semibold tracking-[-0.01em]"
              >
                {current?.title ??
                  topics.at(-1)?.title ??
                  t("dash.finishedTitle")}
              </h2>
              <p className="text-muted-foreground mt-1.5 max-w-[640px] text-sm">
                {current?.goal ?? topics.at(-1)?.goal ?? t("dash.finishedText")}
              </p>
            </div>
            <div className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-[13px]">
              {upcoming && (
                <span>
                  <b className="text-foreground font-medium">
                    {t("dash.nextTopic")}
                  </b>{" "}
                  · {upcoming.title}
                </span>
              )}
              <span>
                <b className="text-foreground font-medium">{plan.length}</b>{" "}
                {t("dash.activities")}
              </span>
              <span>
                <b className="text-foreground font-medium">{lessonMinutes}</b>{" "}
                {t("dash.minutes")}
              </span>
            </div>
            <div className="bg-muted rounded-lg p-3 text-[13px]">
              <b className="font-medium">{t("dash.why")}</b> {why}
            </div>
            <div>
              <div className="flex flex-wrap gap-2">
                <Button asChild size="lg" className="h-10 gap-2 px-4 text-sm">
                  <Link href={continueHref}>
                    <IconPlayerPlay size={16} aria-hidden="true" />
                    {t(finished ? "dash.extraPractice" : "dash.start")}
                  </Link>
                </Button>
                {reviewActivities.length > 0 && (
                  <Button
                    variant="outline"
                    size="lg"
                    className="h-10 px-4 text-sm"
                    onClick={() => setSelectedReview(reviewActivities[0])}
                  >
                    {t("dash.reviewStart")}
                  </Button>
                )}
              </div>
            </div>
          </section>

          <section
            aria-label={t("dash.summary")}
            className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-3"
          >
            <Metric
              icon={<IconClock size={14} aria-hidden="true" />}
              label={t("dash.mWeekly")}
              value={formatStudy(weekSeconds)}
              unit={goalSeconds > 0 ? `/ ${formatStudy(goalSeconds)}` : ""}
              pct={goalSeconds > 0 ? (weekSeconds / goalSeconds) * 100 : 0}
              kind={t("dash.kindActivity")}
            />
            <Metric
              icon={<IconRefresh size={14} aria-hidden="true" />}
              label={t("dash.mReview")}
              value={String(dueCount)}
              unit={t("dash.unitItems")}
              pct={
                reviews && reviews.due + reviews.upcoming > 0
                  ? (reviews.due / (reviews.due + reviews.upcoming)) * 100
                  : 0
              }
              kind={t("dash.kindActivity")}
            />
            <Metric
              icon={<IconCheck size={14} aria-hidden="true" />}
              label={t("dash.mMastered")}
              value={String(items?.mastered ?? 0)}
              unit={t("dash.unitItems")}
              pct={
                items && items.tracked > 0
                  ? (items.mastered / items.tracked) * 100
                  : 0
              }
              kind={t("dash.kindKnowledge")}
              tone="knowledge"
            />
            <Metric
              icon={<IconCalendar size={14} aria-hidden="true" />}
              label={t("dash.mConsistency")}
              value={String(activeDays)}
              unit={t("dash.unitDays")}
              pct={(activeDays / 7) * 100}
              kind={t("dash.kindActivity")}
            />
          </section>

          <section className={card}>
            <div className="border-border flex items-center justify-between gap-3 border-b px-5 py-4">
              <div>
                <h2 className="text-base font-semibold">
                  {t("dash.planTitle")}
                </h2>
                <p className="text-muted-foreground text-xs">
                  {finished
                    ? t("dash.planDone", { done, total: topics.length })
                    : plan.length > 0
                      ? t("dash.planSummary", { done, total: topics.length })
                      : t("dash.planNone")}
                </p>
              </div>
              <Button
                asChild
                variant="link"
                size="sm"
                className="h-auto px-0 text-xs underline"
              >
                <Link href={programHref}>{t("dash.editPlan")}</Link>
              </Button>
            </div>
            {plan.length > 0 && (
              <ol className="p-2">
                {plan.map((row) => (
                  <li
                    key={row.key}
                    className="flex flex-wrap items-center gap-3 rounded-lg p-3"
                  >
                    <div className="min-w-48 flex-1">
                      <p className="text-muted-foreground text-xs">
                        {row.type}
                      </p>
                      <p className="font-medium">{row.title}</p>
                      {row.objective && (
                        <p className="text-muted-foreground text-xs">
                          {t("dash.objective")}: {row.objective}
                        </p>
                      )}
                    </div>
                    <Pill tone={row.tone}>{row.status}</Pill>
                    {row.href && (
                      <Button
                        asChild
                        variant="outline"
                        size="lg"
                        className="h-8 px-3 text-[13px]"
                      >
                        <Link
                          href={row.href}
                          aria-label={t("dash.openRow", { title: row.title })}
                        >
                          {t("dash.open")}
                        </Link>
                      </Button>
                    )}
                    {row.key === "review" && reviewActivities.length > 0 && (
                      <Button
                        variant="outline"
                        size="lg"
                        className="h-8 px-3 text-[13px]"
                        onClick={() => setSelectedReview(reviewActivities[0])}
                      >
                        {t("dash.open")}
                      </Button>
                    )}
                  </li>
                ))}
              </ol>
            )}
            {plan.length === 0 && (
              <p className="text-muted-foreground px-5 py-8 text-center text-sm">
                {t(finished ? "dash.planFinished" : "dash.planNone")}
              </p>
            )}
          </section>

          <WeekChart
            bars={bars}
            weekSeconds={weekSeconds}
            goalSeconds={goalSeconds}
          />
        </div>

        <div className="flex min-w-0 flex-col gap-6">
          <section className={cn(card, "px-5 py-4")}>
            <h2 className="mb-1 text-base font-semibold">
              {t("dash.skillsTitle")}
            </h2>
            <p className="text-muted-foreground mb-2 text-xs">
              {t("dash.skillsHint")}
            </p>
            <ul>
              {skillIds
                .filter((skill) => skill !== "pronunciation")
                .map((skill) => {
                  const inSkill = topics.filter(
                    (topic) => topic.skill === skill,
                  );
                  const finishedInSkill = inSkill.filter(
                    (topic) => topic.status === "completed",
                  ).length;

                  return (
                    <li
                      key={skill}
                      className="border-border flex items-center justify-between gap-3 border-t py-2"
                    >
                      <span className="font-medium">{t(`skill.${skill}`)}</span>
                      <span
                        className={cn(
                          "text-[13px]",
                          inSkill.length === 0 && "text-muted-foreground",
                        )}
                      >
                        {inSkill.length > 0
                          ? t("dash.topicsDone", {
                              done: finishedInSkill,
                              total: inSkill.length,
                            })
                          : t("dash.notInProgram")}
                      </span>
                    </li>
                  );
                })}
            </ul>
            <p className="text-muted-foreground mt-2 text-xs">
              {t("estimatesNote")}
            </p>
          </section>

          <ReviewQueue
            activities={reviewActivities}
            total={dueCount}
            loading={loop.isPending}
            now={now}
            locale={locale}
            name={name(teacher.language)}
            onOpen={setSelectedReview}
          />

          <section className={cn(card, "px-5 py-4")}>
            <div className="mb-2 flex items-center gap-2">
              <IconSparkles size={16} aria-hidden="true" />
              <h2 className="text-base font-semibold">
                {t("dash.insightsTitle")}
              </h2>
            </div>
            <p className="text-muted-foreground text-sm">
              {t("dash.insightsNone")}
            </p>
          </section>
        </div>
      </div>
    </div>
  );
}

type PlanRow = {
  key: string;
  type: string;
  title: string;
  objective: string;
  status: string;
  tone: Tone;
  href: string | null;
};

function Metric({
  icon,
  label,
  value,
  unit,
  pct,
  kind,
  tone = "activity",
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  unit: string;
  pct: number;
  kind: string;
  tone?: "activity" | "knowledge";
}) {
  const width = Math.max(0, Math.min(100, Math.round(pct)));

  return (
    <div className={cn(card, "flex flex-col gap-2 p-4")}>
      <div className="text-muted-foreground flex items-center gap-1.5 text-xs">
        {icon}
        {label}
      </div>
      <div className="text-[22px] font-semibold tracking-[-0.01em]">
        {value}
        {unit && (
          <span className="text-muted-foreground text-[13px] font-normal">
            {" "}
            {unit}
          </span>
        )}
      </div>
      <div
        role="progressbar"
        aria-label={label}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={width}
        className="bg-muted h-1 overflow-hidden rounded-sm"
      >
        <div
          className={cn(
            "h-full",
            tone === "knowledge" ? "bg-success" : "bg-info",
          )}
          style={{ width: `${width}%` }}
        />
      </div>
      <p className="text-muted-foreground text-xs">{kind}</p>
    </div>
  );
}

function WeekChart({
  bars,
  weekSeconds,
  goalSeconds,
}: {
  bars: Bar[];
  weekSeconds: number;
  goalSeconds: number;
}) {
  const t = useTranslations("home");
  const top = Math.max(30, ...bars.map((bar) => bar.minutes));

  return (
    <section className={cn(card, "px-5 py-4")}>
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h2 className="text-base font-semibold">{t("dash.weekTitle")}</h2>
          <p className="text-muted-foreground text-xs">{t("dash.weekHint")}</p>
        </div>
        <div className="text-right">
          <span className="text-xl font-semibold">
            {formatStudy(weekSeconds)}
          </span>
          {goalSeconds > 0 && (
            <span className="text-muted-foreground">
              {" "}
              / {formatStudy(goalSeconds)}
            </span>
          )}
        </div>
      </div>
      <div
        role="img"
        aria-label={t("dash.weekAria")}
        className="border-border mt-4 flex h-[132px] items-end gap-3 border-b"
      >
        {bars.map((bar) => (
          <div
            key={bar.key}
            className="flex h-full flex-1 flex-col items-center justify-end gap-1"
          >
            <span className="text-muted-foreground text-[11px]">
              {bar.minutes || ""}
            </span>
            <div
              className={cn(
                "min-h-0.5 w-full max-w-10 rounded-t",
                bar.today
                  ? "bg-success"
                  : bar.future
                    ? "bg-muted"
                    : "bg-faint/60",
              )}
              style={{ height: `${(bar.minutes / top) * 90}px` }}
            />
          </div>
        ))}
      </div>
      <div className="mt-1.5 flex gap-3">
        {bars.map((bar) => (
          <span
            key={bar.key}
            className={cn(
              "flex-1 text-center text-xs",
              bar.today
                ? "text-foreground font-medium"
                : "text-muted-foreground",
            )}
          >
            {bar.label}
          </span>
        ))}
      </div>
    </section>
  );
}

function ReviewQueue({
  activities,
  total,
  loading,
  now,
  locale,
  name,
  onOpen,
}: {
  activities: LoopActivity[];
  total: number;
  loading: boolean;
  now: number;
  locale: string;
  name: string;
  onOpen: (activity: LoopActivity) => void;
}) {
  const t = useTranslations("home");
  const date = new Intl.DateTimeFormat(locale, {
    day: "numeric",
    month: "short",
  });

  return (
    <section className={card}>
      <div className="flex flex-wrap items-center justify-between gap-2 px-5 pt-4 pb-2">
        <h2 className="text-base font-semibold">{t("dash.reviewTitle")}</h2>
        {activities.length > 0 && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => onOpen(activities[0])}
          >
            {t("dash.reviewStart")}
          </Button>
        )}
      </div>
      {activities.length > 0 ? (
        <ul className="px-5 pb-2">
          {activities.map((activity) => {
            const due = activity.dueAt ? Date.parse(activity.dueAt) : now;
            const overdue = now - due > dayMs;

            return (
              <li key={activity.id} className="border-border border-t py-3">
                <div className="flex items-center justify-between gap-2">
                  <button
                    type="button"
                    className="text-left font-medium underline-offset-2 hover:underline"
                    onClick={() => onOpen(activity)}
                  >
                    {activity.title || activity.item?.key}
                  </button>
                  <Pill tone="review">
                    <IconClock size={12} aria-hidden="true" />
                    {t(overdue ? "dash.overdue" : "dash.dueToday")}
                  </Pill>
                </div>
                <p className="text-muted-foreground text-[13px]">
                  {t(
                    activity.lapsed
                      ? "dash.reasonLapsed"
                      : "dash.reasonInterval",
                  )}
                </p>
                <p className="text-muted-foreground text-xs">
                  {t("dash.dueOn", { date: date.format(due) })} · {name}
                </p>
              </li>
            );
          })}
          {total > activities.length && (
            <li className="border-border text-muted-foreground border-t py-3 text-xs">
              {t("dash.moreDue", { count: total - activities.length })}
            </li>
          )}
        </ul>
      ) : (
        <p className="text-muted-foreground px-5 pt-2 pb-6 text-sm">
          {loading
            ? t("dash.reviewLoading")
            : total > 0
              ? t("dash.reviewUnavailable")
              : t("dash.reviewNone")}
        </p>
      )}
    </section>
  );
}

/** The learner has no AI provider (or lost it): what is saved still works. */
export function AiNotice() {
  const t = useTranslations("home");

  return (
    <div
      role="alert"
      className="border-border bg-muted flex flex-wrap items-start gap-3 rounded-[10px] border px-4 py-3"
    >
      <div className="min-w-[260px] flex-1">
        <p className="font-semibold">{t("ai.title")}</p>
        <p className="text-muted-foreground text-sm">{t("ai.text")}</p>
      </div>
      <Button
        asChild
        variant="outline"
        size="lg"
        className="h-9 px-3 text-[13px]"
      >
        <Link href="/learning/settings">{t("configureAi")}</Link>
      </Button>
    </div>
  );
}
