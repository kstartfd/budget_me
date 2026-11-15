import type { Path } from "./schemas"

/** One topic is one lesson of this many minutes (the program is sized by it). */
export const lessonMinutes = 30

/**
 * Plans the topics that are not completed yet onto calendar days, from `today`:
 * the learner's weekly goal says how many lessons fit in a week, and they are
 * spread evenly over its seven days. It is a guide for pacing, not a deadline,
 * so it is worked out here from what the program already holds and moves with
 * the learner: finish a topic early and everything behind it comes closer.
 *
 * Returns the day of each open topic by its id, and the last day of the plan;
 * nothing when the learner gave no weekly goal.
 */
export function scheduleTopics(
  path: Pick<Path, "topics" | "weeklyGoalMinutes">,
  today: Date,
): { days: Map<string, Date>; finish: Date | null } {
  const perWeek = Math.max(1, Math.round(path.weeklyGoalMinutes / lessonMinutes))
  const days = new Map<string, Date>()

  if (path.weeklyGoalMinutes <= 0) return { days, finish: null }

  const start = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  let finish: Date | null = null
  let n = 0

  for (const topic of path.topics) {
    if (topic.status === "completed") continue

    const week = Math.floor(n / perWeek)
    const offset = Math.floor(((n % perWeek) * 7) / perWeek)
    const day = new Date(start)
    day.setDate(start.getDate() + week * 7 + offset)

    days.set(topic.id, day)
    finish = day
    n++
  }

  return { days, finish }
}
