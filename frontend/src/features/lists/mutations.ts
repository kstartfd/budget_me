"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"

import {
  createItem,
  createList,
  deleteItem,
  deleteList,
  updateItem,
} from "./api"
import { listKeys } from "./queries"

export function useCreateList() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: createList,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: listKeys.dictionary() }),
  })
}

export function useDeleteList() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteList,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: listKeys.dictionary() }),
  })
}

export function useCreateItem(listId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: createItem,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: listKeys.items(listId) }),
  })
}

export function useUpdateItem(listId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: updateItem,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: listKeys.items(listId) }),
  })
}

export function useDeleteItem(listId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteItem,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: listKeys.items(listId) }),
  })
}
