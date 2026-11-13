import { useMemo, useSyncExternalStore } from "react"

/**
 * Where the learner left the teacher's screen for the rule book, so that
 * going back lands them on the same spot: the same teacher, the same open
 * topic, the very link they pressed, which is framed for a moment.
 *
 * It lives in the tab's session storage, not in the address: the browser's own
 * Back button must find it as well as the links of the page, and nothing of it
 * belongs in a shared link.
 */
export type ReturnPoint = {
  /** The page they left, as an address: the Teacher page, or the chat that embeds it. */
  href: string
  language: string
  translationLanguage: string
  /** The topic that was open, if one was. */
  topicId: string | null
  /** What they pressed; the element with this `data-return-anchor` is framed. */
  anchor: string
  at: number
}

const storageKey = "learning.rule-return"

/** An old point is not a way back but a leftover: the tab was left and used for something else. */
const lifetimeMs = 15 * 60_000

/** The anchor of the link to a related rule inside an open topic. */
export const seeAlsoAnchor = (ruleId: string) => `rule-link-${ruleId}`

/** The anchor of the button that opens the rule book from the program. */
export const rulesButtonAnchor = "rules-open"

/** What the framed element carries while it is framed. */
export const flashAttribute = "data-return-flash"

/** How long the frame stays. */
export const flashMs = 4000

export function rememberReturnPoint(point: Omit<ReturnPoint, "href" | "at">) {
  try {
    const stored: ReturnPoint = {
      ...point,
      href: `${window.location.pathname}${window.location.search}`,
      at: Date.now(),
    }
    sessionStorage.setItem(storageKey, JSON.stringify(stored))
  } catch {
    // Storage can be blocked; the learner then goes back the plain way.
  }
}

function readRaw(): string | null {
  try {
    return sessionStorage.getItem(storageKey)
  } catch {
    return null
  }
}

function parse(raw: string | null): ReturnPoint | null {
  if (!raw) return null

  try {
    const point = JSON.parse(raw) as Partial<ReturnPoint>
    if (
      typeof point.href !== "string" ||
      !point.href.startsWith("/learning/") ||
      typeof point.language !== "string" ||
      typeof point.translationLanguage !== "string" ||
      typeof point.anchor !== "string" ||
      typeof point.at !== "number" ||
      Date.now() - point.at > lifetimeMs
    ) {
      return null
    }

    return {
      href: point.href,
      language: point.language,
      translationLanguage: point.translationLanguage,
      topicId: typeof point.topicId === "string" ? point.topicId : null,
      anchor: point.anchor,
      at: point.at,
    }
  } catch {
    return null
  }
}

/** The way back, left where it is. */
export const peekReturnPoint = (): ReturnPoint | null => parse(readRaw())

/** The way back is used up: the learner is where it led. */
export function forgetReturnPoint() {
  try {
    sessionStorage.removeItem(storageKey)
  } catch {
    // Nothing to clear.
  }
}

const never = () => () => {}

/**
 * The way back as a page reads it. The server renders without it and the
 * browser adds it after hydration, so the two never disagree on what is drawn.
 */
export function useReturnPoint(): ReturnPoint | null {
  const raw = useSyncExternalStore(never, readRaw, () => null)

  return useMemo(() => parse(raw), [raw])
}
