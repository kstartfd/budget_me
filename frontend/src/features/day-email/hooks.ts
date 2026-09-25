"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { useAuth } from "@/features/auth"

import {
  confirmDayEmail,
  fetchDayEmailPreference,
  updateDayEmailPreference,
} from "./api"

const key = (id?: string, email?: string) =>
  ["day-email-preference", id ?? null, email ?? null] as const

export function useDayEmailPreference() {
  const { user } = useAuth()
  return useQuery({
    queryKey: key(user?.id, user?.email),
    queryFn: fetchDayEmailPreference,
    enabled: !!user,
  })
}

export function useUpdateDayEmailPreference() {
  const client = useQueryClient()
  const { user } = useAuth()
  return useMutation({
    mutationFn: updateDayEmailPreference,
    onSuccess: (data) => client.setQueryData(key(user?.id, user?.email), data),
  })
}

export function useConfirmDayEmail() {
  const client = useQueryClient()
  const { user } = useAuth()
  return useMutation({
    mutationFn: confirmDayEmail,
    onSuccess: (data) => client.setQueryData(key(user?.id, user?.email), data),
  })
}
