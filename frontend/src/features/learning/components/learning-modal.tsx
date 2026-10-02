"use client"

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog"

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
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent className="top-auto bottom-0 max-h-[92dvh] w-full max-w-full translate-y-0 grid-cols-[minmax(0,1fr)] overflow-y-auto rounded-t-[28px] rounded-b-none bg-card p-5 pb-[max(env(safe-area-inset-bottom),1.25rem)] sm:top-1/2 sm:bottom-auto sm:max-w-lg sm:-translate-y-1/2 sm:rounded-[24px] sm:p-6">
        <DialogTitle className="pr-7 text-lg font-bold tracking-tight">
          {title}
        </DialogTitle>
        <DialogDescription
          className={description ? "text-muted-foreground text-sm" : "sr-only"}
        >
          {description ?? title}
        </DialogDescription>
        {children}
      </DialogContent>
    </Dialog>
  )
}
