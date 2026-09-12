import { shiftDate, todayISO } from "@/lib/month"

import type { Task } from "./schemas"

/**
 * Mock store for the tasks feature — seeded rows in localStorage, standing in
 * for `internal/tasks` until the backend lands. api.ts is the ONLY module that
 * touches it, so wiring the real endpoints means swapping the fetchers and
 * deleting this file; the hooks and screens never learn anything changed.
 */

const storageKey = "budget_me.tasks.v1"

function seed(): Task[] {
  const today = todayISO()
  const at = (delta: number) => shiftDate(today, delta)

  return [
    {
      id: "t-analyses",
      title: "Сдать анализы",
      note: "натощак, до 11:00",
      // Three days old and still open — this is the row that proves the
      // carry-over: nothing moved it, it simply has not been ticked.
      plannedFor: at(-3),
      doneOn: null,
      createdAt: `${at(-3)}T09:12:00.000Z`,
    },
    {
      id: "t-cat-food",
      title: "Купить корм коту",
      note: null,
      plannedFor: at(-1),
      doneOn: null,
      createdAt: `${at(-1)}T18:40:00.000Z`,
    },
    {
      id: "t-hosting",
      title: "Оплатить хостинг",
      note: null,
      plannedFor: today,
      doneOn: null,
      createdAt: `${today}T08:05:00.000Z`,
    },
    {
      id: "t-dentist",
      title: "Записаться к стоматологу",
      note: null,
      plannedFor: today,
      doneOn: null,
      createdAt: `${today}T08:06:00.000Z`,
    },
    {
      id: "t-mail",
      title: "Разобрать почту",
      note: null,
      plannedFor: today,
      doneOn: today,
      createdAt: `${today}T08:07:00.000Z`,
    },
    {
      id: "t-mom",
      title: "Позвонить маме",
      note: null,
      plannedFor: today,
      doneOn: today,
      createdAt: `${today}T08:08:00.000Z`,
    },
  ]
}

function read(): Task[] {
  if (typeof window === "undefined") return seed()

  try {
    const raw = window.localStorage.getItem(storageKey)
    if (!raw) {
      const rows = seed()
      write(rows)

      return rows
    }

    return JSON.parse(raw) as Task[]
  } catch {
    // Private mode / blocked storage: keep the screen working on a fresh seed
    // rather than failing the query.
    return seed()
  }
}

function write(rows: Task[]) {
  if (typeof window === "undefined") return

  try {
    window.localStorage.setItem(storageKey, JSON.stringify(rows))
  } catch {
    // Nothing to do — the in-memory result is still correct for this session.
  }
}

export const mockTasks = {
  all: read,
  save: write,
}
