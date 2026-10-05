"use client"

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog"
import { BottomSheet } from "@/features/budget/components/bottom-sheet"
import { useIsDesktop } from "@/hooks/use-is-desktop"

/**
 * Centered dialog on desktop, real bottom sheet on mobile — same split as
 * ItemSheet, so Learning's forms match the rest of the app instead of
 * rendering as a generic centered shadcn dialog on every size.
 */
export function LearningModal({
  title,
  description,
  children,
  onClose,
}: {
  title: string
  description?: string
  children: React.ReactNode
  onClose: () => void
}) {
  const isDesktop = useIsDesktop()

  if (isDesktop) {
    return (
      <Dialog open onOpenChange={(open) => !open && onClose()}>
        <DialogContent className="bg-card border-border max-h-[85vh] gap-0 overflow-y-auto rounded-[26px] p-6 sm:max-w-lg">
          <DialogTitle className="text-lg font-bold tracking-[-0.02em]">
            {title}
          </DialogTitle>
          <DialogDescription
            className={
              description ? "text-muted-foreground mt-1 text-sm" : "sr-only"
            }
          >
            {description ?? title}
          </DialogDescription>
          <div className="mt-4">{children}</div>
        </DialogContent>
      </Dialog>
    )
  }

  return (
    <BottomSheet title={title} onClose={onClose}>
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
