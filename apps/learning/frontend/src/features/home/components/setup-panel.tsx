"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  IconBook,
  IconClipboardCheck,
  IconPlayerPlay,
} from "@tabler/icons-react";

import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAuth } from "@/features/auth";
import { hasQuickAssessment } from "@/features/assessment/availability";
import * as api from "@/features/path/api";
import { usePathTest } from "@/features/path/hooks";
import type { PathTeacher } from "@/features/path/schemas";
import { cn } from "@/lib/utils";

import {
  claimableLevels,
  defaultProfile,
  preferencesOf,
  useProfile,
  useSetupMemory,
  type Profile,
  type StartId,
} from "../profile";
import type { Stage } from "../setup";
import { useLanguageName } from "./language-switcher";
import { Caption, Choice, Pill, card, type Tone } from "./parts";

const badgeTone: Partial<Record<Stage["kind"], Tone>> = {
  "assess-resume": "learning",
};

/**
 * What to do next for a language that has no program yet: pick where to start,
 * start or resume the assessment, or build the program. Beside it, what the
 * profile already holds. The program itself is built in the background, so the
 * learner can leave while it runs.
 */
export function SetupPanel({
  teacher,
  stage,
  aiReady,
  onOpenTest,
  showProfile = true,
}: {
  teacher: PathTeacher;
  stage: Exclude<
    Stage,
    { kind: "loading" | "active" | "generating" | "failed" | "unavailable" }
  >;
  /** False while the learner has no AI provider: what needs it waits. */
  aiReady: boolean;
  onOpenTest: (testId: string) => void;
  showProfile?: boolean;
}) {
  const t = useTranslations("home");
  const name = useLanguageName();
  const { language, translationLanguage } = teacher;
  const [stored, saveProfile] = useProfile(language);
  const [, saveMemory] = useSetupMemory(language);
  const profile: Profile = stored ?? defaultProfile;
  const build = useBuild(teacher);

  const setStart = (start: StartId | undefined) =>
    saveProfile({ ...profile, start });

  const startTest = useMutation({
    mutationFn: () => api.startPlacement(language, translationLanguage),
    onSuccess: (test) => {
      saveMemory({ testId: test.id });
      onOpenTest(test.id);
    },
  });

  const failure =
    (startTest.isError && startTest.error.message) ||
    (build.isError && build.error.message) ||
    null;

  const copy = {
    choose: {
      title: t("setup.choose.title"),
      text: t("setup.choose.text", { language: name(language) }),
      note: t("setup.choose.note"),
    },
    "assess-ready": {
      title: t("setup.assessReady.title"),
      text: t("setup.assessReady.text"),
      note: t("setup.assessReady.note"),
    },
    "assess-resume": {
      title: t("setup.resume.title"),
      text: t("setup.resume.text", {
        count: stage.kind === "assess-resume" ? stage.answered : 0,
      }),
      note: t("setup.resume.note"),
    },
    "create-manual": {
      title: t("setup.create.title"),
      text: t("setup.create.textManual", { level: profile.level }),
      note: t("setup.create.note"),
    },
    "create-scratch": {
      title: t("setup.create.title"),
      text: t("setup.create.textScratch"),
      note: t("setup.create.note"),
    },
    "create-assessed": {
      title: t("setup.create.title"),
      text:
        stage.kind === "create-assessed"
          ? stored?.start === "scratch"
            ? t("setup.create.textScratch")
            : t(
                stage.claimed
                  ? "setup.create.textClaimed"
                  : "setup.create.textAssessed",
                { level: stage.level },
              )
          : "",
      note: t("setup.create.note"),
    },
  }[stage.kind];

  const badge = t(`setup.badge.${stage.kind}`);
  const quickAssessment =
    stage.kind === "assess-ready" && hasQuickAssessment(language);
  const needsAi = stage.kind !== "choose" && !quickAssessment;

  return (
    <div
      className={
        showProfile
          ? "grid grid-cols-[repeat(auto-fit,minmax(min(100%,380px),1fr))] items-start gap-6"
          : ""
      }
    >
      <section
        id="starting-point"
        className={cn(card, "flex flex-col gap-4 p-6")}
      >
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Caption>{t("nextStep")}</Caption>
          <Pill tone={badgeTone[stage.kind] ?? "neutral"}>{badge}</Pill>
        </div>
        <div>
          <h2 className="text-[22px] font-semibold tracking-[-0.01em]">
            {copy.title}
          </h2>
          <p className="text-muted-foreground mt-1.5 max-w-[560px] text-sm">
            {copy.text}
          </p>
        </div>

        {stage.kind === "choose" && (
          <div className="grid gap-3 sm:grid-cols-3">
            {(["scratch", "manual", "assess"] as const).map((id) => {
              const Icon =
                id === "scratch"
                  ? IconPlayerPlay
                  : id === "manual"
                    ? IconBook
                    : IconClipboardCheck;

              return (
                <Choice
                  key={id}
                  disabled={build.isPending}
                  onClick={() => setStart(id)}
                  className="flex min-h-32 flex-col items-start gap-1.5 p-4"
                >
                  <Icon size={18} aria-hidden="true" />
                  <span className="font-medium">
                    {t(`startOption.${id}.title`)}
                  </span>
                  <span className="text-muted-foreground block text-[13px]">
                    {t(`startOption.${id}.text`)}
                  </span>
                </Choice>
              );
            })}
          </div>
        )}

        {stage.kind === "create-manual" && (
          <label className="block max-w-xs space-y-1 text-[13px]">
            <span>{t("approxLevel")}</span>
            <span className="text-muted-foreground block">
              {t("selfReported")}
            </span>
            <Select
              value={profile.level}
              onValueChange={(level) =>
                saveProfile({ ...profile, level: level as Profile["level"] })
              }
            >
              <SelectTrigger
                className="bg-background h-10 w-full px-3 text-sm"
                aria-label={t("approxLevel")}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {claimableLevels.map((level) => (
                  <SelectItem key={level} value={level}>
                    {level}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>
        )}

        {stage.kind !== "choose" && (
          <div className="flex flex-wrap items-center gap-2">
            {quickAssessment ? (
              <Button asChild size="lg" className="h-11 px-5 text-sm">
                <Link href="/learning/assessment">
                  {t("setup.cta.assess-ready")}
                </Link>
              </Button>
            ) : (
              <Button
                size="lg"
                disabled={
                  (needsAi && !aiReady) ||
                  startTest.isPending ||
                  build.isPending
                }
                onClick={() => {
                  if (stage.kind === "assess-ready") startTest.mutate();
                  else if (stage.kind === "assess-resume")
                    onOpenTest(stage.testId);
                  else if (stage.kind === "create-manual")
                    build.mutate({ kind: "claim", level: profile.level });
                  else if (stage.kind === "create-scratch")
                    build.mutate({ kind: "scratch" });
                  else build.mutate({ kind: "test", testId: stage.testId });
                }}
                className="h-11 px-5 text-sm"
              >
                {t(
                  startTest.isPending || build.isPending
                    ? "working"
                    : `setup.cta.${stage.kind}`,
                )}
              </Button>
            )}
            <Button
              variant="outline"
              size="lg"
              disabled={build.isPending || startTest.isPending}
              onClick={() => {
                saveMemory(null);
                setStart(
                  stage.kind === "assess-resume" ? "scratch" : undefined,
                );
              }}
              className="h-11 px-4 text-sm"
            >
              {t(`setup.secondary.${stage.kind}`)}
            </Button>
          </div>
        )}

        {failure && (
          <p role="alert" className="text-destructive text-[13px]">
            {failure}
          </p>
        )}
        {needsAi && !aiReady && (
          <p className="text-muted-foreground text-xs">{t("needsAi")}</p>
        )}
        <p className="text-muted-foreground text-xs">{copy.note}</p>
      </section>

      {showProfile && <ProfileCard teacher={teacher} stage={stage} />}
    </div>
  );
}

type BuildInput =
  | { kind: "scratch" }
  | { kind: "claim"; level: string }
  | { kind: "test"; testId: string };

/**
 * Starts the building of the program from where the learner is: a test that is
 * already finished, a level they claim, or nothing but the fundamentals. The
 * job runs on the server; this only remembers it, so the screen can follow it.
 */
export function useBuild(teacher: PathTeacher) {
  const { user } = useAuth();
  const client = useQueryClient();
  const { language, translationLanguage } = teacher;
  const [stored] = useProfile(language);
  const [, saveMemory] = useSetupMemory(language);

  return useMutation({
    mutationFn: async (input: BuildInput) => {
      let testId: string;

      if (input.kind === "test") {
        testId = input.testId;
      } else {
        const test = await api.chooseLevel(
          language,
          translationLanguage,
          input.kind === "claim" ? input.level : "A1",
          true,
        );
        testId = test.id;
        saveMemory({ testId });
      }

      const job = await api.startBuild(
        testId,
        preferencesOf(stored ?? defaultProfile),
      );
      saveMemory({ testId, jobId: job.id });
    },
    onSuccess: () => client.invalidateQueries({ queryKey: ["path", user?.id] }),
  });
}

export function ProfileCard({
  teacher,
  stage,
}: {
  teacher: PathTeacher;
  stage: Stage;
}) {
  const t = useTranslations("home");
  const name = useLanguageName();
  const [stored] = useProfile(teacher.language);

  const startMark =
    stage.kind === "create-assessed" ||
    stage.kind === "create-manual" ||
    stage.kind === "create-scratch"
      ? "✓"
      : stage.kind === "assess-resume"
        ? "◐"
        : "○";
  const startText =
    stage.kind === "assess-resume"
      ? t("check.inProgress")
      : stage.kind === "create-scratch" ||
          (stage.kind === "create-assessed" && stored?.start === "scratch")
        ? t("startValue.scratch")
        : stage.kind === "create-assessed"
          ? t(stage.claimed ? "check.claimed" : "check.assessed", {
              level: stage.level,
            })
          : stage.kind === "create-manual"
            ? t("check.claimed", {
                level: stored?.level ?? defaultProfile.level,
              })
            : t("check.notChosen");

  const items: [string, string, string][] = [
    ["✓", t("check.profile"), t("check.profileText")],
    [startMark, t("check.start"), startText],
    ["○", t("check.path"), t("check.notCreated")],
  ];

  const goals = stored?.goals.map((id) => t(`goal.${id}`)).join(", ");
  const skills = stored?.skills.map((id) => t(`skill.${id}`)).join(", ");

  return (
    <section className={cn(card, "flex flex-col gap-4 p-5")}>
      <div>
        <h2 className="text-base font-semibold">
          {t("profileOf", { language: name(teacher.language) })}
        </h2>
        <p className="text-muted-foreground text-xs">{t("teacherSaved")}</p>
      </div>
      <ol className="flex flex-col gap-2 text-sm">
        {items.map(([mark, title, sub]) => (
          <li key={title} className="flex gap-2">
            <span aria-hidden="true" className="w-[18px]">
              {mark}
            </span>
            <span>
              <span className="font-medium">{title}</span>{" "}
              <span className="text-muted-foreground text-[13px]">· {sub}</span>
            </span>
          </li>
        ))}
      </ol>
      <dl className="border-border grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 border-t pt-3 text-[13px]">
        <dt className="text-muted-foreground">{t("sumExplanations")}</dt>
        <dd>{name(teacher.explainLanguage)}</dd>
        {stored && (
          <>
            <dt className="text-muted-foreground">{t("sumGoals")}</dt>
            <dd>{goals || t("noneShort")}</dd>
            <dt className="text-muted-foreground">{t("sumRoutine")}</dt>
            <dd>
              {t("routineValue", {
                hours: stored.weekly,
                minutes: stored.session,
              })}
            </dd>
            <dt className="text-muted-foreground">{t("skillPriorities")}</dt>
            <dd>{skills || t("noneSet")}</dd>
          </>
        )}
      </dl>
      <p className="text-muted-foreground text-xs">{t("estimatesNote")}</p>
    </section>
  );
}

/**
 * The program is being written. Nothing here needs the learner to stay: the
 * job keeps going on the server and the program is there when they come back.
 */
export function GeneratingPanel({ teacher }: { teacher: PathTeacher }) {
  const t = useTranslations("home");
  const name = useLanguageName();
  const language = name(teacher.language);

  return (
    <section aria-busy="true" className="flex max-w-[880px] flex-col gap-4">
      <div className={cn(card, "p-5")}>
        <p className="text-base font-semibold">
          {t("building.title", { language })}
        </p>
        <p className="text-muted-foreground mt-1 mb-2 text-sm">
          {t("building.text")}
        </p>
        <ul className="text-muted-foreground flex flex-col gap-1 text-[13px]">
          <li>✓ {t("building.saved")}</li>
          <li className="text-foreground">● {t("building.selecting")}</li>
          <li>○ {t("building.arranging")}</li>
        </ul>
      </div>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,300px),1fr))] gap-4">
        {["45%", "60%"].map((width) => (
          <div key={width} className={cn(card, "flex flex-col gap-2.5 p-5")}>
            <div
              className="bg-muted h-4 animate-pulse rounded"
              style={{ width }}
            />
            <div className="bg-muted h-3 w-[85%] animate-pulse rounded" />
            <div className="bg-muted h-3 w-[60%] animate-pulse rounded" />
            <div className="bg-muted h-3 w-[70%] animate-pulse rounded" />
          </div>
        ))}
      </div>
    </section>
  );
}

/** The build failed: everything the learner gave is still saved. */
export function FailedPanel({
  teacher,
  testId,
  message,
  aiReady,
}: {
  teacher: PathTeacher;
  testId: string | null;
  message: string;
  aiReady: boolean;
}) {
  const t = useTranslations("home");
  const name = useLanguageName();
  const [stored, saveProfile] = useProfile(teacher.language);
  const [, saveMemory] = useSetupMemory(teacher.language);
  const build = useBuild(teacher);
  const test = usePathTest(!aiReady ? testId : null);
  const canRetry = aiReady || !!test.data?.assessment;

  return (
    <section
      role="alert"
      className={cn(card, "flex max-w-[880px] flex-col gap-3 p-6")}
    >
      <p className="text-base font-semibold">{t("failed.title")}</p>
      <p className="text-muted-foreground max-w-[560px] text-sm">
        {t("failed.text", { language: name(teacher.language) })}
      </p>
      {(message || build.isError) && (
        <p className="text-destructive text-[13px]">
          {build.isError ? build.error.message : message}
        </p>
      )}
      <div className="flex flex-wrap gap-2">
        {testId && (
          <Button
            size="lg"
            disabled={build.isPending || !canRetry}
            onClick={() => build.mutate({ kind: "test", testId })}
            className="h-11 px-5 text-sm"
          >
            {t(build.isPending ? "working" : "retry")}
          </Button>
        )}
        <Button
          variant="outline"
          size="lg"
          disabled={build.isPending}
          onClick={() => {
            saveMemory(null);
            saveProfile({ ...(stored ?? defaultProfile), start: undefined });
          }}
          className="h-11 px-4 text-sm"
        >
          {t("changeStart")}
        </Button>
      </div>
    </section>
  );
}
