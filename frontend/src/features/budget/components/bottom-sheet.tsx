"use client"

import { useEffect } from "react"

/**
 * Design bottom sheet: dim backdrop, rounded panel sliding up, drag handle,
 * title row with a ✕ chip. Content is whatever the caller renders (amount
 * display, chips, numpad). Esc and backdrop close it. An optional `onBack`
 * renders a ‹ chip before the title (the two-step add flow's step 2).
 */
export function BottomSheet({
  title,
  onClose,
  onBack,
  children,
}: {
  title: string
  onClose: () => void
  onBack?: () => void
  children: React.ReactNode
}) {
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose()
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [onClose])

  return (
    <div className="fixed inset-0 z-30 flex flex-col justify-end">
      <div
        aria-hidden
        onClick={onClose}
        className="anim-fade absolute inset-0 bg-[rgba(6,8,14,0.5)]"
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="anim-sheet bg-card relative mx-auto w-full max-w-md rounded-t-[28px] px-5 pt-2.5 pb-[max(env(safe-area-inset-bottom),1.625rem)] shadow-[0_-10px_40px_rgba(0,0,0,0.3)]"
      >
        <div className="bg-faint mx-auto mt-1.5 mb-3.5 h-1 w-9 rounded-full opacity-40" />
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2.5">
            {onBack && (
              <button
                type="button"
                aria-label="‹"
                onClick={onBack}
                className="border-border bg-secondary text-foreground flex size-[30px] items-center justify-center rounded-full border text-base leading-none"
              >
                ‹
              </button>
            )}
            <span className="text-lg font-bold tracking-[-0.02em]">
              {title}
            </span>
          </div>
          <button
            type="button"
            aria-label="✕"
            onClick={onClose}
            className="bg-secondary text-muted-foreground size-[30px] rounded-full text-base"
          >
            ✕
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}
