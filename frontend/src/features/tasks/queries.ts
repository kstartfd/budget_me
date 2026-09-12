"use client"

import { useQuery } from "@tanstack/react-query"

import { fetchDayTasks } from "./api"

/**
 * Read hooks for the tasks feature. Keyed per day: moving between days is a
 * different list, and a postpone therefore has to invalidate the whole
 * feature prefix (the task leaves one day and joins another).
 */

export const taskKeys = {
  all: ["tasks"] as const,
  day: (date: string) => [...taskKeys.all, "day", date] as const,
}

export function useDayTasks(date: string) {
  return useQuery({
    queryKey: taskKeys.day(date),
    queryFn: () => fetchDayTasks(date),
  })
}
