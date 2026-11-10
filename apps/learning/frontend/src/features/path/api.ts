import { readToken } from "@/features/auth"
import {
  pathAttemptSchema,
  pathExerciseSchema,
  pathLessonSchema,
  pathSchema,
  pathTestSchema,
  pathTopicDetailSchema,
  pathTopicSchema,
  type Path,
  type PathAttempt,
  type PathExercise,
  type PathLesson,
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

export const startPlacement = (language: string, translationLanguage: string): Promise<PathTest> =>
  request(
    "/path/tests",
    "POST",
    (d) => pathTestSchema.parse(d),
    { language, translationLanguage },
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

/** Builds the program from a finished placement test. */
export const analyzeTest = (
  id: string,
  preferences: Preferences = {},
): Promise<Path> =>
  request(
    `/path/tests/${id}/analyze`,
    "POST",
    (d) => pathSchema.parse(d),
    preferences,
  )

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
