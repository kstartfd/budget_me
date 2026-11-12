import type { Path, PathTopic } from "./schemas"

/**
 * How long after a program is built its exercises may still be on their way:
 * the server writes them in the background, a few at a time, and gives up on
 * the rest after this long (the learner's topic then writes its own set).
 */
const preparingWindowMs = 25 * 60 * 1000

/** After this much of the window the screen asks less often. */
const quickPollMs = 3 * 60 * 1000

/**
 * How far along one topic is, 0 to 1, by its exercises: a completed topic is
 * whole, a topic whose set is not written yet is nothing, and the rest is the
 * share of its current set answered right. A regenerated set starts over, and
 * so does the share.
 */
export function topicProgress(topic: PathTopic): number {
  if (topic.status === "completed") return 1

  const { total, done } = topic.exercises
  if (total <= 0) return 0

  return Math.min(done / total, 1)
}

/** Whether the topic still waits for its exercise set to be written. */
export const awaitsExercises = (topic: PathTopic) =>
  topic.status !== "completed" && topic.exercises.total === 0

/**
 * Whether the program's exercises may still be being written: it is fresh and
 * some topic has no set yet. The screen shows a note and keeps asking while so.
 */
export function exercisesPreparing(path: Path, now: number): boolean {
  if (path.status !== "active") return false
  if (!path.topics.some(awaitsExercises)) return false

  return now - Date.parse(path.createdAt) < preparingWindowMs
}

/** How long to wait before asking a fresh program again; false when it is settled. */
export function preparingPollMs(path: Path | null | undefined, now: number) {
  if (!path || !exercisesPreparing(path, now)) return false

  return now - Date.parse(path.createdAt) < quickPollMs ? 5_000 : 15_000
}
