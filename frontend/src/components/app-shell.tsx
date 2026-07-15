"use client"

import { AppHeader } from "@/components/app-header"
import { BottomNav } from "@/components/bottom-nav"
import { DesktopNav } from "@/components/desktop-nav"
import { DeletionBanner } from "@/features/account"
import { AuthGate, AuthProvider } from "@/features/auth"
import { AddSheet } from "@/features/budget/components/add-sheet"
import { BalanceSheet } from "@/features/budget/components/balance-sheet"
import { LimitSheet } from "@/features/budget/components/limit-sheet"
import { PlanSheet } from "@/features/budget/components/plan-sheet"
import { BudgetToast } from "@/features/budget/components/toast"
import { TxDetailSheet } from "@/features/budget/components/tx-detail-sheet"
import { BudgetUIProvider } from "@/features/budget/components/ui-context"

/**
 * Shared frame for every screen. Desktop (lg+) is a left sidebar + main column;
 * phone / Telegram keep the sticky AppHeader, a single centered column and the
 * fixed BottomNav (whose center is the "add expense" button). Both share the
 * sheets and toast overlays any screen can summon via useBudgetUI().
 *
 * Everything sits behind the AuthGate — without a session (Telegram initData
 * or a web login) visitors only ever see the auth screen.
 */
export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <AuthGate>
        <BudgetUIProvider>
          <div className="lg:flex">
            <DesktopNav />
            <div className="min-w-0 flex-1">
              <AppHeader />
              <main className="mx-auto flex min-h-dvh w-full max-w-md flex-col gap-4 px-[18px] pt-[18px] pb-44 lg:max-w-4xl lg:px-6 lg:pt-6 lg:pb-16">
                <DeletionBanner />
                {children}
              </main>
            </div>
            <BottomNav />
            <AddSheet />
            <BalanceSheet />
            <LimitSheet />
            <PlanSheet />
            <TxDetailSheet />
            <BudgetToast />
          </div>
        </BudgetUIProvider>
      </AuthGate>
    </AuthProvider>
  )
}
