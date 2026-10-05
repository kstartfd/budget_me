"use client"

import { BottomSheet } from "@/features/budget/components/bottom-sheet"

/**
 * Always a bottom sheet — same choice as AddSheet (the expense add flow),
 * not the desktop-dialog/mobile-sheet split ItemSheet uses. Learning's forms
 * are simple single-column stacks with no reason to re-layout for width.
 */
export function LearningModal({
  title,
  description,
  children,
  onClose,
  onBack,
}: {
  title: string
  description?: string
  children: React.ReactNode
  onClose: () => void
  onBack?: () => void
}) {
  return (
    <BottomSheet title={title} onClose={onClose} onBack={onBack}>
      <div className="max-h-[calc(100dvh-7rem)] overflow-y-auto pt-4">
        {description && (
          <p className="text-muted-foreground -mt-1 mb-3 text-sm leading-relaxed">
            {description}
          </p>
        )}
        {children}
      </div>
    </BottomSheet>
  )
}
