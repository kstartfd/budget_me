"use client"

import { wash } from "./wash"
import { useBudgetUI, type ToastTone } from "./ui-context"

/**
 * Top slide-down toast (limit saved / removed / validation nudge). One at a
 * time, auto-dismissed by the provider after 4.6s.
 */

const toneColor: Record<ToastTone, string> = {
  info: "#5b8def",
  success: "#35b98f",
  danger: "#ea6a6a",
}

const toneIcon: Record<ToastTone, string> = {
  info: "ℹ",
  success: "✓",
  danger: "⚠",
}

export function BudgetToast() {
  const { toast } = useBudgetUI()
  if (!toast) return null
  const color = toneColor[toast.tone]
  return (
    <div className="pointer-events-none fixed inset-x-0 top-3 z-40">
      <div className="mx-auto w-full max-w-md px-3">
        <div
          role="status"
          aria-live="polite"
          className="anim-toast bg-card border-border flex items-center gap-[11px] rounded-2xl border px-[13px] py-[11px] shadow-[0_16px_36px_rgba(0,0,0,0.34)]"
        >
          <span
            className="flex size-[30px] shrink-0 items-center justify-center rounded-[10px] text-[15px] font-bold"
            style={{ background: wash(color), color }}
          >
            {toneIcon[toast.tone]}
          </span>
          <span className="flex-1 text-[12.5px] leading-[1.4] font-medium">
            {toast.text}
          </span>
        </div>
      </div>
    </div>
  )
}
