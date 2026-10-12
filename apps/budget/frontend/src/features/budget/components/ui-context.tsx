"use client"

import {
  createContext,
  useCallback,
  useContext,
  useRef,
  useState,
} from "react"

import { useSettings } from "@/features/settings"
import { formatFx } from "@/lib/money"

import { useRates } from "../queries"
import type { Plan, RateSource, Transaction } from "../schemas"

/**
 * Screen-level UI state shared across routes: the add-expense sheet (opened
 * by the floating button in the shell), the limit sheet (opened from the
 * Limits screen and from AI insight CTAs), the top toast, and the cycling
 * rate source. Pure UI state — server data stays in TanStack Query.
 *
 * Currency: amounts are always in Br. The setting picks what they are shown
 * *next to* — with BYN there is nothing to convert into, so `rate` is null and
 * `fx()` returns null, and every "≈ …" line and the rate pill disappear.
 */

export type ToastTone = "info" | "success" | "danger"

interface LimitSheetState {
  categoryId: string
  /** Pre-filled limit in minor units (AI suggestion or current limit), if any. */
  prefillMinor: number | null
}

/** Plan sheet target — a null plan means "new plan", otherwise edit that plan. */
interface PlanSheetState {
  plan: Plan | null
}

/** Balance sheet: `prefill` seeds the numpad when correcting a known balance. */
interface BalanceSheetState {
  prefill: string | null
}

interface BudgetUIContextValue {
  addOpen: boolean
  openAdd: () => void
  closeAdd: () => void
  /** Transaction being edited — reuses the add sheet in edit mode when set. */
  editTx: Transaction | null
  openEdit: (tx: Transaction) => void
  /** Transaction whose detail sheet is open (full comment + actions). */
  detailTx: Transaction | null
  openDetail: (tx: Transaction) => void
  closeDetail: () => void
  limitSheet: LimitSheetState | null
  openLimit: (categoryId: string, prefillMinor?: number | null) => void
  closeLimit: () => void
  planSheet: PlanSheetState | null
  openPlan: (plan?: Plan | null) => void
  closePlan: () => void
  balanceSheet: BalanceSheetState | null
  openBalance: (prefill?: string | null) => void
  closeBalance: () => void
  toast: { id: number; tone: ToastTone; text: string } | null
  showToast: (tone: ToastTone, text: string) => void
  /** Active rate source (Br per unit of the chosen currency); tapping the
   *  dashboard pill cycles it. Null when the currency is BYN. */
  rate: RateSource | null
  cycleRate: () => void
  /** "≈ $44" companion for a Br amount — null when no conversion applies, so
   *  callers render the line only when there is one. */
  fx: (minor: number) => string | null
}

const BudgetUIContext = createContext<BudgetUIContextValue | null>(null)

const fallbackRate: RateSource = { slug: "nbrb", rate: 3.27 }
const toastMs = 4600

export function BudgetUIProvider({ children }: { children: React.ReactNode }) {
  const [addOpen, setAddOpen] = useState(false)
  const [editTx, setEditTx] = useState<Transaction | null>(null)
  const [detailTx, setDetailTx] = useState<Transaction | null>(null)
  const [limitSheet, setLimitSheet] = useState<LimitSheetState | null>(null)
  const [planSheet, setPlanSheet] = useState<PlanSheetState | null>(null)
  const [balanceSheet, setBalanceSheet] = useState<BalanceSheetState | null>(
    null,
  )
  const [toast, setToast] = useState<BudgetUIContextValue["toast"]>(null)
  const [rateIdx, setRateIdx] = useState(0)
  const toastTimer = useRef<number | null>(null)

  const settings = useSettings()
  const currency = settings.data?.currency ?? "BYN"
  const converts = currency !== "BYN"

  const rates = useRates(currency)
  const rateList = rates.data?.length ? rates.data : [fallbackRate]
  const rate = converts ? rateList[rateIdx % rateList.length] : null

  const fx = useCallback(
    (minor: number) =>
      rate ? formatFx(minor, rate.rate, currency) : null,
    [rate, currency],
  )

  const showToast = useCallback((tone: ToastTone, text: string) => {
    if (toastTimer.current) window.clearTimeout(toastTimer.current)
    const id = Date.now()
    setToast({ id, tone, text })
    toastTimer.current = window.setTimeout(() => {
      setToast((t) => (t && t.id === id ? null : t))
    }, toastMs)
  }, [])

  const value: BudgetUIContextValue = {
    addOpen,
    openAdd: useCallback(() => {
      setEditTx(null)
      setAddOpen(true)
    }, []),
    closeAdd: useCallback(() => {
      setAddOpen(false)
      setEditTx(null)
    }, []),
    editTx,
    openEdit: useCallback((tx: Transaction) => setEditTx(tx), []),
    detailTx,
    openDetail: useCallback((tx: Transaction) => setDetailTx(tx), []),
    closeDetail: useCallback(() => setDetailTx(null), []),
    limitSheet,
    openLimit: useCallback(
      (categoryId: string, prefillMinor: number | null = null) =>
        setLimitSheet({ categoryId, prefillMinor }),
      [],
    ),
    closeLimit: useCallback(() => setLimitSheet(null), []),
    planSheet,
    openPlan: useCallback((plan: Plan | null = null) => setPlanSheet({ plan }), []),
    closePlan: useCallback(() => setPlanSheet(null), []),
    balanceSheet,
    openBalance: useCallback(
      (prefill: string | null = null) => setBalanceSheet({ prefill }),
      [],
    ),
    closeBalance: useCallback(() => setBalanceSheet(null), []),
    toast,
    showToast,
    rate,
    cycleRate: useCallback(
      () => setRateIdx((i) => (i + 1) % rateList.length),
      [rateList.length],
    ),
    fx,
  }

  return (
    <BudgetUIContext.Provider value={value}>
      {children}
    </BudgetUIContext.Provider>
  )
}

export function useBudgetUI(): BudgetUIContextValue {
  const ctx = useContext(BudgetUIContext)
  if (!ctx) throw new Error("useBudgetUI must be used inside BudgetUIProvider")
  return ctx
}
