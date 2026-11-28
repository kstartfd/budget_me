"use client"

import { z } from "zod"

import { useAuth } from "@/features/auth"
import type { Preferences } from "@/features/path/schemas"

import { useStored } from "./store"

export const setupSteps = ["native", "language", "goals", "routine", "start"] as const

/** What the learner says they want, in the order the wizard offers it. */
export const goalIds = [
  "everyday",
  "professional",
  "travel",
  "relocation",
  "academic",
  "exam",
  "interest",
] as const
export type GoalId = (typeof goalIds)[number]

export const skillIds = [
  "grammar",
  "vocabulary",
  "reading",
  "listening",
  "speaking",
  "writing",
  "pronunciation",
] as const
export type SkillId = (typeof skillIds)[number]

export const prefIds = ["shortGrammar", "moreListening", "spacedReminders"] as const
export type PrefId = (typeof prefIds)[number]

export const startIds = ["assess", "manual", "scratch"] as const
export type StartId = (typeof startIds)[number]

export const weeklyOptions = [2, 3, 5, 8] as const
export const sessionOptions = [15, 30, 45, 60] as const
/** The self-reported levels offered: the ladder without its top rung. */
export const claimableLevels = ["A1", "A2", "B1", "B2", "C1"] as const

/**
 * The answers of the setup wizard for one language. The backend keeps the
 * language, how it is explained and the weekly hours of the program; the rest is
 * kept here until it has a place there, and is what the program is planned from.
 */
export const profileSchema = z.object({
  goals: z.array(z.enum(goalIds)),
  skills: z.array(z.enum(skillIds)),
  weekly: z.number().int().min(1).max(40),
  session: z.number().int().min(5).max(180),
  prefs: z.array(z.enum(prefIds)),
  /** Where the learner chose to start; absent until they have chosen. */
  start: z.enum(startIds).optional(),
  level: z.enum(claimableLevels),
})
export type Profile = z.infer<typeof profileSchema>

export const defaultProfile: Profile = {
  goals: ["everyday", "travel"],
  skills: [],
  weekly: 3,
  session: 30,
  prefs: ["shortGrammar", "spacedReminders"],
  start: "assess",
  level: "A2",
}

const parseProfile = (raw: unknown) => {
  const parsed = profileSchema.safeParse(raw)

  return parsed.success ? parsed.data : null
}

/** How far the setup of a language has got, so a closed tab does not lose it. */
export const setupSchema = z.object({
  /** The placement (or the self-reported level) the program is built from. */
  testId: z.string().optional(),
  /** The background build of the program. */
  jobId: z.string().optional(),
})
export type SetupMemory = z.infer<typeof setupSchema>

const parseSetup = (raw: unknown) => {
  const parsed = setupSchema.safeParse(raw)

  return parsed.success ? parsed.data : null
}

const parseText = (raw: unknown) => {
  const parsed = z.string().min(1).max(35).safeParse(raw)

  return parsed.success ? parsed.data : null
}

const prefix = "learning_app.home.v1"

/** The wizard's answers for a language, or null when it was set up without the wizard. */
export function useProfile(language: string | null) {
  const { user } = useAuth()
  const key = user && language ? `${prefix}.profile.${user.id}.${language}` : null

  return useStored(key, parseProfile)
}

export function useSetupMemory(language: string | null) {
  const { user } = useAuth()
  const key = user && language ? `${prefix}.setup.${user.id}.${language}` : null

  return useStored(key, parseSetup)
}

/** The language on screen: the learner's last pick, or their first teacher. */
export function useActiveLanguage() {
  const { user } = useAuth()
  const key = user ? `${prefix}.active.${user.id}` : null

  return useStored(key, parseText)
}

/** English names: they go into the prompt the program is planned from, not onto a screen. */
const goalWords: Record<GoalId, string> = {
  everyday: "everyday communication",
  professional: "professional communication",
  travel: "travel",
  relocation: "relocation",
  academic: "academic study",
  exam: "exam preparation",
  interest: "personal interest",
}

const prefWords: Record<PrefId, string> = {
  shortGrammar: "short grammar explanations",
  moreListening: "more listening practice",
  spacedReminders: "spaced review reminders",
}

/** The survey the program is planned around, made from the wizard's answers. */
export function preferencesOf(profile: Profile): Preferences {
  const interests = [
    ...profile.skills.map((skill) => `priority skill: ${skill}`),
    ...profile.prefs.map((pref) => prefWords[pref]),
  ].join("; ")

  return {
    goal: profile.goals.map((goal) => goalWords[goal]).join(", ") || undefined,
    interests: interests || undefined,
    hoursPerWeek: profile.weekly,
  }
}
