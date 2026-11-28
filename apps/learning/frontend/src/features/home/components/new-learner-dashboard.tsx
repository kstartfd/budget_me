"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import {
  IconCalendar,
  IconCheck,
  IconClock,
  IconRefresh,
  IconSparkles,
} from "@tabler/icons-react";

import { Button } from "@/components/ui/button";
import type { PathTeacher } from "@/features/path/schemas";
import { cn } from "@/lib/utils";

import { skillIds } from "../profile";
import type { Stage } from "../setup";
import { card } from "./parts";
import { ProfileCard, SetupPanel } from "./setup-panel";

type SetupStage = Exclude<
  Stage,
  { kind: "loading" | "active" | "generating" | "failed" | "unavailable" }
>;

/** Overview before the language has a program. All summary blocks are honest
 * empty states; the existing setup state machine owns the start actions. */
export function NewLearnerDashboard({
  teacher,
  stage,
  aiReady,
  onOpenTest,
}: {
  teacher: PathTeacher;
  stage: SetupStage;
  aiReady: boolean;
  onOpenTest: (testId: string) => void;
}) {
  const t = useTranslations("home");
  const estimate =
    stage.kind === "create-assessed"
      ? t("dash.estimatePending")
      : t("dash.notAssessed");

  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,420px),1fr))] items-start gap-6">
      <div className="flex min-w-0 flex-col gap-6">
        <SetupPanel
          teacher={teacher}
          stage={stage}
          aiReady={aiReady}
          onOpenTest={onOpenTest}
          showProfile={false}
        />
        <section
          aria-label={t("dash.summary")}
          className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-3"
        >
          <EmptyMetric icon={IconClock} label={t("dash.mWeekly")} />
          <EmptyMetric icon={IconRefresh} label={t("dash.mReview")} />
          <EmptyMetric icon={IconCheck} label={t("dash.mMastered")} />
          <EmptyMetric icon={IconCalendar} label={t("dash.mConsistency")} />
        </section>
        <section className={card}>
          <div className="border-border border-b px-5 py-4">
            <h2 className="text-base font-semibold">{t("dash.planTitle")}</h2>
            <p className="text-muted-foreground text-xs">
              {t("dash.planNotCreated")}
            </p>
          </div>
          <p className="text-muted-foreground px-5 py-8 text-center text-sm">
            {t("dash.planPending")}
          </p>
        </section>
        <section className={cn(card, "px-5 py-4")}>
          <div className="flex items-start justify-between gap-3">
            <div>
              <h2 className="text-base font-semibold">{t("dash.weekTitle")}</h2>
              <p className="text-muted-foreground text-xs">
                {t("dash.weekHint")}
              </p>
            </div>
            <span className="text-muted-foreground text-xs">
              {t("dash.noData")}
            </span>
          </div>
          <div
            className="border-border mt-4 h-32 border-b"
            aria-label={t("dash.noData")}
          />
        </section>
      </div>
      <div className="flex min-w-0 flex-col gap-6">
        <section className={cn(card, "px-5 py-4")}>
          <h2 className="text-base font-semibold">{t("dash.skillsTitle")}</h2>
          <p className="text-muted-foreground mb-2 text-xs">
            {t("dash.skillsEvidenceHint")}
          </p>
          <ul>
            {skillIds.map((skill) => (
              <li
                key={skill}
                className="border-border flex items-center justify-between gap-3 border-t py-2.5"
              >
                <span className="font-medium">{t(`skill.${skill}`)}</span>
                <span className="flex items-center gap-3">
                  <span className="flex gap-0.5" aria-hidden="true">
                    {[0, 1, 2].map((n) => (
                      <span
                        key={n}
                        className="bg-border h-1 w-3 rounded-full"
                      />
                    ))}
                  </span>
                  <span className="text-muted-foreground min-w-24 text-right text-xs">
                    {estimate}
                  </span>
                </span>
              </li>
            ))}
          </ul>
          <p className="text-muted-foreground mt-2 text-xs">
            {t("estimatesNote")}
          </p>
          {stage.kind === "create-assessed" && (
            <Button
              asChild
              variant="link"
              className="mt-2 h-auto px-0 text-xs underline"
            >
              <Link href="/learning/assessment">
                {t("dash.viewAssessment")}
              </Link>
            </Button>
          )}
        </section>
        <section className={cn(card, "px-5 py-4")}>
          <h2 className="text-base font-semibold">{t("dash.reviewTitle")}</h2>
          <p className="text-muted-foreground mt-4 text-sm">
            {t("dash.reviewNotStarted")}
          </p>
        </section>
        <section className={cn(card, "px-5 py-4")}>
          <h2 className="flex items-center gap-2 text-base font-semibold">
            <IconSparkles size={16} aria-hidden="true" />
            {t("dash.insightsTitle")}
          </h2>
          <p className="text-muted-foreground mt-4 text-sm">
            {t("dash.insightsPending")}
          </p>
        </section>
        <ProfileCard teacher={teacher} stage={stage} />
      </div>
    </div>
  );
}

function EmptyMetric({
  icon: Icon,
  label,
}: {
  icon: typeof IconClock;
  label: string;
}) {
  const t = useTranslations("home.dash");

  return (
    <div className={cn(card, "flex flex-col gap-2 p-4")}>
      <p className="text-muted-foreground flex items-center gap-1.5 text-xs">
        <Icon size={14} aria-hidden="true" />
        {label}
      </p>
      <b className="text-[22px]">–</b>
      <div className="bg-muted h-1 rounded-sm" />
      <p className="text-muted-foreground text-xs">{t("noData")}</p>
    </div>
  );
}
