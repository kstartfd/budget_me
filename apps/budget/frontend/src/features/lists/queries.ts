"use client"

import { useQuery } from "@tanstack/react-query"

import { fetchListItems, fetchLists } from "./api"

export const listKeys = {
  all: ["lists"] as const,
  dictionary: () => [...listKeys.all, "dictionary"] as const,
  items: (listId: string) => [...listKeys.all, "items", listId] as const,
}

const dictionaryStaleMs = 10 * 60 * 1000

export function useLists() {
  return useQuery({
    queryKey: listKeys.dictionary(),
    queryFn: fetchLists,
    staleTime: dictionaryStaleMs,
  })
}

export function useListItems(listId: string) {
  return useQuery({
    queryKey: listKeys.items(listId),
    queryFn: () => fetchListItems(listId),
  })
}
