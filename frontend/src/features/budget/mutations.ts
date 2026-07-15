"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"

import {
  confirmPlan,
  createCategory,
  createPlan,
  createTransaction,
  deleteBudget,
  deleteCategory,
  deletePlan,
  deleteTransaction,
  saveBudget,
  setBalance,
  updateCategory,
  updatePlan,
  updateTransaction,
} from "./api"
import { budgetKeys } from "./queries"
import type { Category } from "./schemas"

/**
 * Write hooks. A write invalidates the whole ["budget"] prefix minus the two
 * things it cannot have changed: the exchange rates (a public feed, same for
 * everyone) and the category dictionary (only the category mutations below
 * touch it, and they invalidate it themselves). Everything else — list,
 * summary, trend, budgets, plans, insights, recommendations — really can move
 * on any write, and the datasets are small enough that refetching them beats
 * tracking fine-grained dependencies.
 */

/** Query-key sections a write can never affect. */
const writeIndependent = new Set<string>(["rates", "categories"])

function useInvalidatingMutation<TArgs, TResult>(
  mutationFn: (args: TArgs) => Promise<TResult>,
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn,
    onSuccess: () =>
      queryClient.invalidateQueries({
        predicate: (query) => {
          const [root, section] = query.queryKey as string[]
          return root === budgetKeys.all[0] && !writeIndependent.has(section)
        },
      }),
  })
}

export function useAddTransaction() {
  return useInvalidatingMutation(createTransaction)
}

export function useUpdateTransaction() {
  return useInvalidatingMutation(updateTransaction)
}

/**
 * Create a category. Persists on the real backend (both directions — `kind`
 * becomes the row's direction). Refetch the one categories list on success (a
 * new category has no transactions, so nothing else needs invalidating).
 */
export function useCreateCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createCategory,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: budgetKeys.categories() }),
  })
}

/**
 * Rename/recolour a category. Optimistic against the list cache so the row
 * updates instantly, then reconciled with the real backend PUT on settle (the
 * settle invalidate refetches the list, so a failed edit reverts).
 */
export function useUpdateCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: updateCategory,
    onMutate: async (variables) => {
      const key = budgetKeys.categories()
      await queryClient.cancelQueries({ queryKey: key })
      const previous = queryClient.getQueryData<Category[]>(key)
      queryClient.setQueryData<Category[]>(key, (old) =>
        (old ?? []).map((c) =>
          c.id === variables.id
            ? { ...c, name: variables.name, color: variables.color }
            : c,
        ),
      )
      return { key, previous }
    },
    onError: (_error, _variables, context) => {
      if (context?.previous) {
        queryClient.setQueryData(context.key, context.previous)
      }
    },
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: budgetKeys.categories() }),
  })
}

/**
 * Delete a personal category. Optimistic removal from the list cache, then
 * reconciled with the real backend DELETE on settle (see api.deleteCategory).
 * The settle invalidate refetches the list, so a failed delete (e.g. a preset
 * → 404) reappears.
 */
export function useDeleteCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteCategory,
    onMutate: async (variables) => {
      const key = budgetKeys.categories()
      await queryClient.cancelQueries({ queryKey: key })
      const previous = queryClient.getQueryData<Category[]>(key)
      queryClient.setQueryData<Category[]>(key, (old) =>
        (old ?? []).filter((c) => c.id !== variables.id),
      )
      return { key, previous }
    },
    onError: (_error, _variables, context) => {
      if (context?.previous) {
        queryClient.setQueryData(context.key, context.previous)
      }
    },
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: budgetKeys.categories() }),
  })
}

export function useDeleteTransaction() {
  return useInvalidatingMutation(deleteTransaction)
}

/** State the balance; re-anchors it (see api.setBalance). */
export function useSetBalance() {
  return useInvalidatingMutation(setBalance)
}

export function useSaveBudget() {
  return useInvalidatingMutation(saveBudget)
}

export function useCreatePlan() {
  return useInvalidatingMutation(createPlan)
}

export function useUpdatePlan() {
  return useInvalidatingMutation(updatePlan)
}

export function useConfirmPlan() {
  return useInvalidatingMutation(confirmPlan)
}

export function useDeletePlan() {
  return useInvalidatingMutation(deletePlan)
}

export function useDeleteBudget() {
  return useInvalidatingMutation(
    (args: { categoryId: string; month: string }) =>
      deleteBudget(args.categoryId, args.month),
  )
}

