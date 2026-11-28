"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { useRouter, useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { Button } from "@/components/ui/button"
import { useAuth } from "@/features/auth"
import {
  LanguageSwitcher,
  useLanguageName,
} from "@/features/home/components/language-switcher"
import { defaultProfile, useProfile } from "@/features/home/profile"
import { useActiveTeacher } from "@/features/home/setup"
import { useStored } from "@/features/home/store"
import * as api from "@/features/path/api"
import type { PathTeacher, PathTest } from "@/features/path/schemas"

import { hasQuickAssessment } from "./availability"
import { AssessmentPlan } from "./plan"
import {
  AssessmentLoading,
  AssessmentRecovery,
  AssessmentResume,
} from "./progress"
import { AssessmentQuestion } from "./question"
import { AssessmentResults } from "./results"
import {
  AssessmentHeading,
  AssessmentIntro,
  AssessmentStart,
  type StartOption,
} from "./start"

const parseId = (value: unknown) =>
  typeof value === "string" && value ? value : null

const grammarProgress = (test: PathTest) =>
  test.assessment?.sections?.find((section) => section.skill === "grammar") ??
  test.assessment

export function AssessmentScreen() {
  const t = useTranslations("assessment")
  const { active, teachers } = useActiveTeacher()

  if (teachers.isError && !active) {
    return (
      <AssessmentRecovery
        title={t("title")}
        error={teachers.error.message}
        onRetry={() => void teachers.refetch()}
      />
    )
  }
  if (teachers.isPending) return <AssessmentLoading title={t("loading")} />
  if (!active) {
    return (
      <div className="space-y-4">
        <AssessmentHeading title={t("title")} description={t("noProfile")} />
        <Button asChild className="h-11 px-4 text-sm">
          <Link href="/learning/start">{t("createProfile")}</Link>
        </Button>
      </div>
    )
  }

  return <AssessmentFlow key={active.id} teacher={active} />
}

type View = "default" | "select" | "intro" | "questions" | "plan" | "manualPlan"
type Change =
  | { kind: "start" }
  | { kind: "action"; id: string; action: api.AssessmentAction }
  | { kind: "answer"; id: string; key: string; answer: string }

function AssessmentFlow({ teacher }: { teacher: PathTeacher }) {
  const params = useSearchParams()
  const router = useRouter()
  const t = useTranslations("assessment")
  const name = useLanguageName()
  const client = useQueryClient()
  const { user } = useAuth()
  const { language, translationLanguage } = teacher
  const languageName = name(language)
  const [stored, saveProfile] = useProfile(language)
  const profile = stored ?? defaultProfile
  const [id, remember] = useStored(
    user ? `learning_app.assessment.${user.id}.${teacher.id}` : null,
    parseId,
  )
  const [view, setView] = useState<View>(params.get("newPlan") === "1" ? "select" : "default")
  // The restart marker is one-time navigation. Clear it so reloading later can
  // resume an unfinished assessment instead of reopening the selection screen.
  useEffect(() => {
    if (params.get("newPlan") === "1") router.replace("/learning/assessment")
  }, [params, router])
  const quickAvailable = hasQuickAssessment(language)
  const [option, setOption] = useState<StartOption>(
    quickAvailable ? "quick" : "manual",
  )
  const [level, setLevel] = useState(profile.level)
  const [resumedFrom, setResumedFrom] = useState<number | null>(null)
  const queryKey = ["assessment", user?.id, id] as const

  const verifyProfile = (test: PathTest) => {
    if (test.assessment?.learnerProfileId !== teacher.id)
      throw new Error(t("unavailableError"))

    return test
  }
  const state = useQuery({
    queryKey,
    queryFn: async () => verifyProfile(await api.fetchTest(id!)),
    enabled: !!id,
    retry: false,
  })

  const change = useMutation({
    mutationFn: async (input: Change) => {
      if (input.kind === "start")
        return verifyProfile(
          await api.startAssessment(language, translationLanguage),
        )
      if (input.kind === "action")
        return verifyProfile(await api.changeAssessment(input.id, input.action))

      return verifyProfile(
        await api.answerBatch(input.id, [
          {
            key: input.key,
            answer: input.answer,
            skip: input.answer === "",
          },
        ]),
      )
    },
    onMutate: () => client.cancelQueries({ queryKey }),
    onSuccess: (test, input) => {
      client.setQueryData(["assessment", user?.id, test.id], test)
      if (input.kind === "action" && input.action === "cancel") {
        if (test.assessment?.status === "cancelled") {
          remember(null)
          setView("select")
        } else {
          setView("default")
        }
        return
      }
      remember(test.id)
      if (
        input.kind === "start" ||
        (input.kind === "action" && input.action === "resume")
      ) {
        setResumedFrom(grammarProgress(test)?.answered ?? null)
        setView(
          grammarProgress(test)?.status === "in_progress" ? "questions" : "default",
        )
      } else if (input.kind === "action") {
        setView("default")
      }
    },
    onError: () => {
      if (id) void state.refetch()
    },
  })

  const choose = () => {
    change.reset()
    setView("select")
  }
  const test = state.data
  const progress = test?.assessment
  const section = test ? grammarProgress(test) : undefined
  const pending = change.isPending
  const action = (next: api.AssessmentAction) => {
    if (test && !pending)
      change.mutate({ kind: "action", id: test.id, action: next })
  }
  const pathUrl = `/learning/path?language=${encodeURIComponent(language)}`
  const failure = change.error?.message
  const question = test?.questions[0]

  if (
    view === "questions" &&
    test &&
    section?.status === "in_progress" &&
    question
  ) {
    return (
      <div className="anim-fade space-y-5">
        <LanguageSwitcher className="lg:hidden" />
        <AssessmentHeading title={t("title")} />
        <AssessmentQuestion
          key={question.key}
          question={question}
          language={languageName}
          locale={language}
          answered={section.answered}
          resumedFrom={resumedFrom}
          busy={pending}
          error={failure}
          onAnswer={(answer) => {
            if (!pending)
              change.mutate({
                kind: "answer",
                id: test.id,
                key: question.key,
                answer,
              })
          }}
          onPause={() => action("pause")}
        />
      </div>
    )
  }

  let content
  if (view === "manualPlan" || (view === "plan" && progress?.result)) {
    content = (
      <AssessmentPlan
        teacher={teacher}
        test={view === "plan" ? test : undefined}
        onBack={view === "plan" ? () => setView("default") : choose}
      />
    )
  } else if (view === "intro" && !teacher.path) {
    content = (
      <AssessmentIntro
        language={languageName}
        onBack={choose}
        onStart={() => {
          if (!pending) change.mutate({ kind: "start" })
        }}
        busy={pending}
        error={failure}
      />
    )
  } else if (view !== "select" && id && state.isPending) {
    content = <AssessmentLoading title={t("loading")} />
  } else if (view !== "select" && id && state.isError && !test) {
    const gone =
      state.error instanceof api.ApiError && state.error.status === 404
    content = (
      <AssessmentRecovery
        title={t("title")}
        error={state.error.message}
        onRetry={() => void state.refetch()}
        onChoose={
          gone
            ? () => {
                remember(null)
                choose()
              }
            : undefined
        }
      />
    )
  } else if (
    view !== "select" &&
    (section?.status === "completed" || section?.status === "partial") &&
    progress?.result
  ) {
    content = (
      <AssessmentResults
        result={progress.result}
        language={language}
        languageName={languageName}
        partial={section.status === "partial"}
        onContinue={() => setView("plan")}
        onChoose={teacher.path ? undefined : choose}
      />
    )
  } else if (
    view !== "select" &&
    (section?.status === "in_progress" || section?.status === "paused")
  ) {
    content = (
      <AssessmentResume
        language={languageName}
        answered={section.answered}
        paused={section.status === "paused"}
        pending={pending}
        onAction={action}
        error={failure}
      />
    )
  } else if (
    view !== "select" &&
    (section?.status === "created" ||
      section?.status === "evaluating" ||
      section?.status === "failed")
  ) {
    content = pending ? (
      <AssessmentLoading title={t("preparing")} />
    ) : (
      <AssessmentRecovery
        title={t(section.status === "failed" ? "failed" : "pendingTitle")}
        error={failure ?? t("pendingText", { count: section.answered })}
        onRetry={() => {
          if (section.status === "failed") setView("intro")
          else action("resume")
        }}
        onChoose={choose}
      />
    )
  } else if (teacher.path) {
    content = (
      <div className="space-y-4">
        <AssessmentHeading title={t("title")} description={t("activePath")} />
        <Button asChild className="h-11 px-4 text-sm">
          <Link href={pathUrl}>{t("openPath")}</Link>
        </Button>
      </div>
    )
  } else {
    content = (
      <AssessmentStart
        option={option}
        level={level}
        onLevel={setLevel}
        onOption={setOption}
        quickAvailable={quickAvailable}
        onContinue={() => {
          if (option === "quick") {
            saveProfile({ ...profile, start: "assess" })
            setView("intro")
          } else if (option === "manual" || option === "scratch") {
            saveProfile({ ...profile, start: option, level })
            setView("manualPlan")
          }
        }}
      />
    )
  }

  return (
    <div className="anim-fade space-y-5">
      <LanguageSwitcher className="lg:hidden" />
      {content}
    </div>
  )
}
