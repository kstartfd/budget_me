/**
 * Public surface of the budget feature. Other features and app routes import
 * from here only — internals (api) stay private.
 */

export {
  type AddTransactionInput,
  type Balance,
  type Budget,
  type Category,
  type CreateCategoryInput,
  type DeleteCategoryInput,
  type Direction,
  type Insight,
  type Insights,
  type MonthKey,
  type MonthSummary,
  type Plan,
  type PlanSuggestion,
  type RateSource,
  type Transaction,
  type TrendPoint,
  type TxTotals,
  type UpdateCategoryInput,
} from "./schemas"

export {
  budgetKeys,
  useBalance,
  useBudgets,
  useCategories,
  useInsights,
  useLimitRecommendations,
  useMonthSummary,
  usePlans,
  usePlanSuggestions,
  useRates,
  useTransactions,
  useTransactionsRange,
  useTransactionTotals,
  useTrend,
} from "./queries"

export {
  useAddTransaction,
  useConfirmPlan,
  useCreateCategory,
  useCreatePlan,
  useDeleteBudget,
  useDeleteCategory,
  useDeletePlan,
  useDeleteTransaction,
  useSaveBudget,
  useSetBalance,
  useUpdateCategory,
  useUpdatePlan,
  useUpdateTransaction,
} from "./mutations"

export { isDuplicateNameError } from "./api"

export { useBudgetUI, BudgetUIProvider } from "./components/ui-context"
export { BottomSheet } from "./components/bottom-sheet"
export { wash, washStrong } from "./components/wash"
