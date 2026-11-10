"use client"

import { BottomSheet } from "@/components/ui/bottom-sheet"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useIsDesktop } from "@/hooks/use-is-desktop"

/**
 * A centered dialog on wide screens, a bottom sheet on phones. Forms and
 * confirmations stay simple single-column stacks, so only the container
 * changes with the viewport.
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
  const desktop = useIsDesktop()

  if (!desktop)
    return (
      <BottomSheet title={title} onClose={onClose} onBack={onBack}>
        <div className="max-h-[calc(100dvh-7rem)] overflow-y-auto pt-4">
          {description && (
            <p className="text-muted-foreground -mt-1 mb-3 text-[13px] leading-relaxed">
              {description}
            </p>
          )}
          {children}
        </div>
      </BottomSheet>
    )

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[calc(100dvh-4rem)] gap-3 overflow-y-auto rounded-lg p-6 text-sm sm:max-w-[600px]">
        <DialogHeader>
          <div className="flex items-center gap-2.5">
            {onBack && (
              <button
                type="button"
                aria-label="‹"
                onClick={onBack}
                className="border-border text-foreground hover:bg-muted flex size-7 items-center justify-center rounded-md border text-base leading-none"
              >
                ‹
              </button>
            )}
            <DialogTitle className="text-base font-semibold">{title}</DialogTitle>
          </div>
          {description ? (
            <DialogDescription className="text-muted-foreground text-[13px] leading-relaxed">
              {description}
            </DialogDescription>
          ) : (
            <DialogDescription className="sr-only">{title}</DialogDescription>
          )}
        </DialogHeader>
        {children}
      </DialogContent>
    </Dialog>
  )
}
