import type { Path, PathTopic } from "./schemas"

/**
 * How long after a program is built its exercises may still be on their way:
 * the server writes them in the background, a few at a time, and gives up on
 * the rest after this long (the learner's topic then writes its own set).
 */
const preparingWindowMs = 25 * 60 * 1000

/**
 * The first minute or two after a program is built, its first topics are being
 * written: a learner who starts one of them then waits for the set instead of
 * asking for a second.
 */
const headStartMs = 90 * 1000

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

  return pollEvery(path, now)
}

/**
 * Whether a topic opened right after its program was built may still get its
 * set from the background writer: it has none yet and the program is fresh.
 */
export function topicSetComing(
  path: Path,
  exercises: number,
  now: number,
): boolean {
  if (exercises > 0 || path.status !== "active") return false

  return now - Date.parse(path.createdAt) < preparingWindowMs
}

/** Whether the background writer is certainly still on its first sets. */
export const writerHasHeadStart = (path: Path, now: number) =>
  now - Date.parse(path.createdAt) < headStartMs

/** How long to wait before asking a fresh topic again; false when it is settled. */
export function topicPollMs(
  detail: { path: Path; exercises: unknown[] } | null | undefined,
  now: number,
) {
  if (!detail || !topicSetComing(detail.path, detail.exercises.length, now)) {
    return false
  }

  return pollEvery(detail.path, now)
}

const pollEvery = (path: Path, now: number) =>
  now - Date.parse(path.createdAt) < quickPollMs ? 5_000 : 15_000
