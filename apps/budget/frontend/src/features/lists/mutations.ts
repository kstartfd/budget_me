"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"

import {
  createItem,
  createList,
  deleteItem,
  deleteItemPhoto,
  deleteList,
  updateItem,
  updateList,
  uploadItemPhoto,
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

export function useUpdateList() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: updateList,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: listKeys.dictionary() }),
  })
}

export function useDeleteList() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteList,
    onSuccess: (_data, id) => {
      queryClient.removeQueries({ queryKey: listKeys.items(id) })
      queryClient.invalidateQueries({ queryKey: listKeys.dictionary() })
    },
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

export function useUploadItemPhoto(listId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, photo }: { id: string; photo: File }) =>
      uploadItemPhoto(id, photo),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: listKeys.items(listId) }),
  })
}

export function useDeleteItemPhoto(listId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteItemPhoto,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: listKeys.items(listId) }),
  })
}
