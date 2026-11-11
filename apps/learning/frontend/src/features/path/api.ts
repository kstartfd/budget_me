import { readToken } from "@/features/auth"
import {
  pathAttemptSchema,
  pathJobSchema,
  pathExerciseSchema,
  pathLessonSchema,
  pathSchema,
  pathTeacherSchema,
  pathTestSchema,
  pathTopicDetailSchema,
  pathTopicSchema,
  type Path,
  type PathAttempt,
  type PathJob,
  type PathExercise,
  type PathLesson,
  type PathTeacher,
  type PathTest,
  type PathTopic,
  type PathTopicDetail,
  type Preferences,
} from "./schemas"

/** ApiError carries the HTTP status so a 404 can mean "none yet". */
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message)
  }
}

async function request<T>(
  path: string,
  method: string,
  parse: (data: unknown) => T,
  data?: unknown,
): Promise<T> {
  const token = readToken()
  const res = await fetch(`/api/v1/learning${path}`, {
    method,
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...(data !== undefined ? { body: JSON.stringify(data) } : {}),
  })
  const body = await res.json().catch(() => null)
  if (!res.ok || body?.code !== 0)
    throw new ApiError(res.status, body?.message ?? res.statusText)
  return parse(body.data)
}

/** level starts the test above A1: the rung after a confirmed final test. */
export const startPlacement = (
  language: string,
  translationLanguage: string,
  level?: string,
): Promise<PathTest> =>
  request(
    "/path/tests",
    "POST",
    (d) => pathTestSchema.parse(d),
    { language, translationLanguage, ...(level ? { level } : {}) },
  )

export const startFinalTest = (language: string, translationLanguage: string): Promise<PathTest> =>
  request(
    "/path/final-tests",
    "POST",
    (d) => pathTestSchema.parse(d),
    { language, translationLanguage },
  )

export const fetchTest = (id: string): Promise<PathTest> =>
  request(`/path/tests/${id}`, "GET", (d) => pathTestSchema.parse(d))

export const answerBatch = (
  id: string,
  answers: { key: string; answer: string }[],
): Promise<PathTest> =>
  request(`/path/tests/${id}/answers`, "POST", (d) => pathTestSchema.parse(d), {
    answers,
  })

/**
 * Queues the building of the program from a finished placement test. It
 * answers at once with a job: the agent works in the background, and asking
 * again hands the same job back.
 */
export const startBuild = (
  id: string,
  preferences: Preferences = {},
): Promise<PathJob> =>
  request(
    `/path/tests/${id}/analyze`,
    "POST",
    (d) => pathJobSchema.parse(d),
    preferences,
  )

/** How a build is going; a finished one carries its program. */
export const fetchJob = (id: string): Promise<PathJob> =>
  request(`/path/jobs/${id}`, "GET", (d) => pathJobSchema.parse(d))

/** The active program of a language, or null when there is none. */
export const fetchActivePath = async (language: string): Promise<Path | null> => {
  try {
    return await request(`/path?language=${language}`, "GET", (d) =>
      pathSchema.parse(d),
    )
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null
    throw error
  }
}

export const fetchTeachers = (): Promise<PathTeacher[]> =>
  request("/path/teachers", "GET", (d) => pathTeacherSchema.array().parse(d))

/** Adds a language to study; a language that has a teacher hands it back. */
export const createTeacher = (
  language: string,
  translationLanguage: string,
): Promise<PathTeacher> =>
  request(
    "/path/teachers",
    "POST",
    (d) => pathTeacherSchema.parse(d),
    { language, translationLanguage },
  )

/** The voice of a word, a sentence to listen to or a phrase to repeat. */
export async function fetchPathAudio(path: string): Promise<Blob> {
  const token = readToken()
  const res = await fetch(`/api/v1/learning${path}`, {
    cache: "no-store",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    throw new ApiError(res.status, body?.message ?? res.statusText)
  }

  return res.blob()
}

/** Removes a teacher; its running program goes to the history. */
export const deleteTeacher = (id: string): Promise<void> =>
  request(`/path/teachers/${id}`, "DELETE", () => undefined)

export const fetchHistory = (): Promise<Path[]> =>
  request("/path/history", "GET", (d) => pathSchema.array().parse(d))

export const resetPath = (language: string): Promise<void> =>
  request("/path/reset", "POST", () => undefined, { language })

export const fetchTopicDetail = (id: string): Promise<PathTopicDetail> =>
  request(`/path/topics/${id}`, "GET", (d) => pathTopicDetailSchema.parse(d))

export const generateExercises = (
  topicId: string,
  regenerate = false,
): Promise<PathExercise[]> =>
  request(
    `/path/topics/${topicId}/exercises${regenerate ? "?regenerate=1" : ""}`,
    "POST",
    (d) => pathExerciseSchema.array().parse(d),
  )

export const submitAttempt = (
  exerciseId: string,
  answer: string,
): Promise<PathAttempt> =>
  request(
    `/path/exercises/${exerciseId}/attempt`,
    "POST",
    (d) => pathAttemptSchema.parse(d),
    { answer },
  )

export const completeTopic = (topicId: string): Promise<PathTopic> =>
  request(`/path/topics/${topicId}/complete`, "POST", (d) =>
    pathTopicSchema.parse(d),
  )

export const startLesson = (topicId: string): Promise<PathLesson> =>
  request(`/path/topics/${topicId}/lessons`, "POST", (d) =>
    pathLessonSchema.parse(d),
  )

/** The running lesson of the learner, or null when none is running. */
export const fetchActiveLesson = (): Promise<PathLesson | null> =>
  request("/path/lessons/active", "GET", (d) =>
    d === null ? null : pathLessonSchema.parse(d),
  )

export const fetchLessons = (topicId: string): Promise<PathLesson[]> =>
  request(`/path/topics/${topicId}/lessons`, "GET", (d) =>
    pathLessonSchema.array().parse(d),
  )

export const changeLesson = (
  id: string,
  action: "pause" | "resume" | "finish",
): Promise<PathLesson> =>
  request(`/path/lessons/${id}/${action}`, "POST", (d) =>
    pathLessonSchema.parse(d),
  )
