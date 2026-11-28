"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useAiSettings } from "@/features/ai";
import { usePath } from "@/features/path/hooks";
import type { PathTeacher } from "@/features/path/schemas";
import { TestRunner } from "@/features/path/components/test-runner";

import { useSetupMemory } from "../profile";
import { useActiveTeacher, useSetupStage, type Stage } from "../setup";
import { AiNotice, Dashboard } from "./dashboard";
import { LanguageSwitcher, useLanguageName } from "./language-switcher";
import { NewLearnerDashboard } from "./new-learner-dashboard";
import { NoTeachers } from "./no-teachers";
import { FailedPanel, GeneratingPanel, useBuild } from "./setup-panel";

/**
 * The learner's home. What it shows depends on how far the language on screen
 * has got: nothing yet, a profile waiting for a starting point, an assessment, a
 * program being written, or a running program. Every one of those is read from
 * the server, with this browser only remembering the last step taken.
 */
export function OverviewScreen() {
  const t = useTranslations("home");
  const router = useRouter();
  const resume = useSearchParams().get("resume");
  const { teachers, list, active } = useActiveTeacher();
  const stage = useSetupStage(active);
  const ai = useAiSettings();
  const aiReady = ai.data != null;
  const [memory] = useSetupMemory(active?.language ?? null);
  const [picked, setPicked] = useState<{
    language: string;
    testId: string;
  } | null>(null);
  const [dismissed, setDismissed] = useState(false);
  const testId =
    picked && picked.language === active?.language
      ? picked.testId
      : resume && (resume === active?.language || resume === "1") && !dismissed
        ? (memory?.testId ?? null)
        : null;

  const close = () => {
    setPicked(null);
    setDismissed(true);
    if (resume) router.replace("/learning/overview");
  };

  if (testId && active) {
    return (
      <Runner key={testId} teacher={active} testId={testId} onClose={close} />
    );
  }

  return (
    <div className="anim-fade space-y-6">
      <div className="lg:hidden">
        <LanguageSwitcher />
      </div>
      <Head
        teacher={active}
        stage={stage}
        none={teachers.isSuccess && list.length === 0}
      />

      {teachers.isPending ? (
        <Loading />
      ) : teachers.isError ? (
        <div className="space-y-3">
          <p role="alert" className="text-destructive text-sm">
            {teachers.error.message}
          </p>
          <Button
            variant="outline"
            disabled={teachers.isFetching}
            onClick={() => void teachers.refetch()}
          >
            {t(teachers.isFetching ? "working" : "retry")}
          </Button>
        </div>
      ) : !active ? (
        <NoTeachers />
      ) : (
        <Body
          key={active.id}
          teacher={active}
          stage={stage}
          aiReady={aiReady}
          onOpenTest={(testId) =>
            setPicked({ language: active.language, testId })
          }
        />
      )}
    </div>
  );
}

function Loading() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-40 w-full rounded-xl" />
      <Skeleton className="h-56 w-full rounded-xl" />
    </div>
  );
}

function Head({
  teacher,
  stage,
  none,
}: {
  teacher: PathTeacher | null;
  stage: Stage;
  none: boolean;
}) {
  const t = useTranslations("home");
  const name = useLanguageName();
  const active = stage.kind === "active";
  const language = teacher ? name(teacher.language) : "";

  const title = none ? t("noneTitle") : t("overviewTitle");
  const subtitle = none
    ? t("noneSubtitle")
    : !teacher || active
      ? t("overviewSubtitle")
      : t("newLearnerSubtitle", { language });

  return (
    <header className="flex flex-wrap items-start justify-between gap-3">
      <div>
        <p className="text-muted-foreground mb-1 text-xs">
          {t("crumb", { page: t("overview") })}
        </p>
        <h1 className="text-2xl font-semibold tracking-[-0.01em]">{title}</h1>
        <p className="text-muted-foreground mt-0.5 text-sm">{subtitle}</p>
      </div>
      {teacher && active && (
        <span className="border-border bg-card inline-flex items-center gap-2 rounded-lg border px-2.5 py-1.5 text-[13px]">
          {t("languageChip", {
            language,
            explain: name(teacher.explainLanguage),
          })}
          <Link
            href={`/learning/path?language=${teacher.language}`}
            className="text-muted-foreground underline"
          >
            {t("languageSettings")}
          </Link>
        </span>
      )}
    </header>
  );
}

function Body({
  teacher,
  stage,
  aiReady,
  onOpenTest,
}: {
  teacher: PathTeacher;
  stage: Stage;
  aiReady: boolean;
  onOpenTest: (testId: string) => void;
}) {
  const t = useTranslations("home");

  switch (stage.kind) {
    case "loading":
      return <Loading />;
    case "active":
      return <ActiveDashboard teacher={teacher} aiReady={aiReady} />;
    case "generating":
      return (
        <>
          {!aiReady && <AiNotice />}
          <GeneratingPanel teacher={teacher} />
        </>
      );
    case "failed":
      return (
        <>
          {!aiReady && <AiNotice />}
          <FailedPanel
            teacher={teacher}
            testId={stage.testId}
            message={stage.message}
            aiReady={aiReady}
          />
        </>
      );
    case "unavailable":
      return (
        <div className="space-y-3">
          <p role="alert" className="text-destructive text-sm">
            {t("setupUnavailable")}
          </p>
          <Button
            variant="outline"
            disabled={stage.retrying}
            onClick={stage.retry}
          >
            {t(stage.retrying ? "working" : "retry")}
          </Button>
        </div>
      );
    default:
      return (
        <>
          {!aiReady && <AiNotice />}
          <NewLearnerDashboard
            teacher={teacher}
            stage={stage}
            aiReady={aiReady}
            onOpenTest={onOpenTest}
          />
        </>
      );
  }
}

function ActiveDashboard({
  teacher,
  aiReady,
}: {
  teacher: PathTeacher;
  aiReady: boolean;
}) {
  const t = useTranslations("home");
  const path = usePath(teacher.language, teacher.translationLanguage).active;

  if (path.isPending) return <Loading />;

  if (!path.data) {
    return (
      <p role="alert" className="text-muted-foreground text-sm">
        {t("programMissing")}
      </p>
    );
  }

  return <Dashboard teacher={teacher} path={path.data} aiReady={aiReady} />;
}

/**
 * The assessment, full screen. When it ends the program is built from the
 * answers given in the wizard, so the learner is not asked the survey twice.
 */
function Runner({
  teacher,
  testId,
  onClose,
}: {
  teacher: PathTeacher;
  testId: string;
  onClose: () => void;
}) {
  const t = useTranslations("home");
  const build = useBuild(teacher);
  const [handed, setHanded] = useState(false);

  if (handed) {
    if (build.isError) {
      return (
        <div role="alert" className="max-w-[560px] space-y-3 py-10">
          <p className="text-destructive text-sm">{build.error.message}</p>
          <Button onClick={onClose} size="lg" className="px-4 text-[13px]">
            {t("goToOverview")}
          </Button>
        </div>
      );
    }

    return <GeneratingPanel teacher={teacher} />;
  }

  return (
    <TestRunner
      id={testId}
      language={teacher.language}
      onClose={onClose}
      onFinished={(test) => {
        setHanded(true);
        build.mutate({ kind: "test", testId: test.id }, { onSuccess: onClose });
      }}
    />
  );
}
