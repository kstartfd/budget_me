"use client"

import { useQuery } from "@tanstack/react-query"

import {
  fetchBalance,
  fetchBudgets,
  fetchCategories,
  fetchInsights,
  fetchLimitRecommendations,
  fetchMonthSummary,
  fetchPlans,
  fetchPlanSuggestions,
  fetchRates,
  fetchTransactions,
  fetchTransactionsRange,
  fetchTrend,
} from "./api"

/**
 * Read hooks for the budget feature. Query keys are namespaced under
 * ["budget", ...] so mutations can invalidate the whole feature with one
 * prefix when a write changes derived data (summary, budgets, trend and the
 * list all depend on transactions).
 */

export const budgetKeys = {
  all: ["budget"] as const,
  categories: () => [...budgetKeys.all, "categories"] as const,
  balance: () => [...budgetKeys.all, "balance"] as const,
  transactions: (month: string) =>
    [...budgetKeys.all, "transactions", month] as const,
  transactionsRange: (from: string, to: string) =>
    [...budgetKeys.all, "transactions", "range", from, to] as const,
  budgets: (month: string) => [...budgetKeys.all, "budgets", month] as const,
  plans: (month: string) => [...budgetKeys.all, "plans", month] as const,
  planSuggestions: (month: string) =>
    [...budgetKeys.all, "plans", "suggestions", month] as const,
  summary: (month: string) => [...budgetKeys.all, "summary", month] as const,
  trend: () => [...budgetKeys.all, "trend"] as const,
  rates: (currency: string) => [...budgetKeys.all, "rates", currency] as const,
  insights: () => [...budgetKeys.all, "insights"] as const,
  limitRecs: () => [...budgetKeys.all, "limit-recs"] as const,
}

const dictionaryStaleMs = 10 * 60 * 1000

export function useCategories() {
  return useQuery({
    queryKey: budgetKeys.categories(),
    queryFn: fetchCategories,
    staleTime: dictionaryStaleMs,
  })
}

export function useBalance() {
  return useQuery({
    queryKey: budgetKeys.balance(),
    queryFn: fetchBalance,
  })
}

export function useTransactions(month: string) {
  return useQuery({
    queryKey: budgetKeys.transactions(month),
    queryFn: () => fetchTransactions(month),
  })
}

export function useTransactionsRange(from: string, to: string) {
  return useQuery({
    queryKey: budgetKeys.transactionsRange(from, to),
    queryFn: () => fetchTransactionsRange(from, to),
  })
}

export function useBudgets(month: string) {
  return useQuery({
    queryKey: budgetKeys.budgets(month),
    queryFn: () => fetchBudgets(month),
  })
}

export function usePlans(month: string) {
  return useQuery({
    queryKey: budgetKeys.plans(month),
    queryFn: () => fetchPlans(month),
  })
}

export function usePlanSuggestions(month: string) {
  return useQuery({
    queryKey: budgetKeys.planSuggestions(month),
    queryFn: () => fetchPlanSuggestions(month),
  })
}

export function useMonthSummary(month: string) {
  return useQuery({
    queryKey: budgetKeys.summary(month),
    queryFn: () => fetchMonthSummary(month),
  })
}

export function useTrend() {
  return useQuery({ queryKey: budgetKeys.trend(), queryFn: fetchTrend })
}

/** Rates for `currency`; idle for BYN — the amounts are already in rubles, so
 *  there is nothing to convert and no reason to hit the providers. */
export function useRates(currency: string) {
  return useQuery({
    queryKey: budgetKeys.rates(currency),
    queryFn: () => fetchRates(currency),
    enabled: currency !== "BYN",
    staleTime: dictionaryStaleMs,
  })
}

export function useInsights() {
  return useQuery({
    queryKey: budgetKeys.insights(),
    queryFn: fetchInsights,
    staleTime: dictionaryStaleMs,
  })
}

export function useLimitRecommendations() {
  return useQuery({
    queryKey: budgetKeys.limitRecs(),
    queryFn: fetchLimitRecommendations,
    staleTime: dictionaryStaleMs,
  })
}
