"use client"

import { useEffect } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"

import { useAuth } from "@/features/auth"
import { ApiError, fetchJob } from "@/features/path/api"
import { usePathTest, useTeachers } from "@/features/path/hooks"
import type { PathTeacher } from "@/features/path/schemas"

import { useActiveLanguage, useProfile, useSetupMemory } from "./profile"

/**
 * Where the setup of a language has got to. A language with a running program
 * is simply `active`; before that the learner is somewhere between choosing how
 * to start and waiting for the program to be built.
 */
export type Stage =
  | { kind: "loading" }
  | { kind: "active" }
  | { kind: "choose" }
  | { kind: "assess-ready" }
  | { kind: "assess-resume"; testId: string; answered: number }
  | { kind: "create-manual" }
  | { kind: "create-scratch" }
  | { kind: "create-assessed"; testId: string; level: string; claimed: boolean }
  | { kind: "generating"; jobId: string; testId: string | null }
  | { kind: "failed"; message: string; testId: string | null }
  | { kind: "unavailable"; retry: () => void; retrying: boolean }

const buildPollMs = 1500

/**
 * The stage of one teacher's setup, read from what the server knows (the
 * program, the test, the build job) and from what this browser remembers of the
 * learner's last step. A job that finished is waited out until the teacher shows
 * its program, so the screen never falls back to "choose" for a moment.
 */
export function useSetupStage(teacher: PathTeacher | null): Stage {
  const { user } = useAuth()
  const client = useQueryClient()
  const language = teacher?.language ?? null
  const [memory, setMemory] = useSetupMemory(language)
  const [profile] = useProfile(language)
  const hasProgram = !!teacher?.path

  const test = usePathTest(!hasProgram && memory?.testId && !memory.jobId ? memory.testId : null)
  const job = useQuery({
    queryKey: ["path", user?.id, "job", memory?.jobId ?? null],
    queryFn: () => fetchJob(memory!.jobId!),
    enabled: !hasProgram && !!memory?.jobId,
    refetchInterval: (query) =>
      query.state.status === "error" ||
      query.state.data?.status === "done" ||
      query.state.data?.status === "failed"
        ? false
        : buildPollMs,
  })

  const jobStatus = job.data?.status
  const testGone = test.error instanceof ApiError && test.error.status === 404

  // A program exists: whatever the setup remembered is spent.
  useEffect(() => {
    if (hasProgram && memory) setMemory(null)
  }, [hasProgram, memory, setMemory])

  // A finished build: read the teachers again, and only then forget the job.
  useEffect(() => {
    if (jobStatus !== "done" || hasProgram) return

    void client.invalidateQueries({ queryKey: ["path", user?.id] })
  }, [jobStatus, hasProgram, client, user?.id])

  // A test the server no longer has cannot be resumed.
  useEffect(() => {
    if (testGone && memory?.testId && !memory.jobId) setMemory(null)
  }, [testGone, memory, setMemory])

  if (!teacher) return { kind: "loading" }
  if (hasProgram) return { kind: "active" }

  if (memory?.jobId) {
    if (job.isError) {
      if (job.error instanceof ApiError && job.error.status === 404)
        return { kind: "failed", message: job.error.message, testId: memory.testId ?? null }

      return { kind: "unavailable", retry: () => void job.refetch(), retrying: job.isFetching }
    }
    if (jobStatus === "failed")
      return { kind: "failed", message: job.data?.error ?? "", testId: memory.testId ?? null }

    return { kind: "generating", jobId: memory.jobId, testId: memory.testId ?? null }
  }

  if (memory?.testId) {
    if (test.isError && !testGone)
      return { kind: "unavailable", retry: () => void test.refetch(), retrying: test.isFetching }

    if (!test.data) return { kind: "loading" }

    if (test.data.status === "in_progress")
      return { kind: "assess-resume", testId: memory.testId, answered: test.data.asked }

    return {
      kind: "create-assessed",
      testId: memory.testId,
      level: test.data.levelReached,
      claimed: test.data.questions.length === 0 && test.data.asked === 0,
    }
  }

  if (profile?.start === "assess") return { kind: "assess-ready" }
  if (profile?.start === "manual") return { kind: "create-manual" }
  if (profile?.start === "scratch") return { kind: "create-scratch" }

  return { kind: "choose" }
}

/** What a language is waiting for, in a word, without asking the server anything. */
export type StatusKey =
  | "program"
  | "generating"
  | "inProgress"
  | "readyToAssess"
  | "readyToCreate"
  | "choose"

export function useTeacherStatus(teacher: PathTeacher): StatusKey {
  const [memory] = useSetupMemory(teacher.language)
  const [profile] = useProfile(teacher.language)

  if (teacher.path) return "program"
  if (memory?.jobId) return "generating"
  if (memory?.testId) return "inProgress"
  if (profile?.start === "assess") return "readyToAssess"
  if (profile?.start === "manual" || profile?.start === "scratch") return "readyToCreate"

  return "choose"
}

/**
 * The teacher on screen. The learner's last pick wins while that teacher still
 * exists; otherwise it is the first one. Null while the teachers load or when
 * there are none.
 */
export function useActiveTeacher() {
  const { teachers } = useTeachers()
  const [stored, setStored] = useActiveLanguage()
  const list = teachers.data ?? []
  const active = list.find((teacher) => teacher.language === stored) ?? list[0] ?? null

  return { teachers, list, active, setActive: setStored }
}
