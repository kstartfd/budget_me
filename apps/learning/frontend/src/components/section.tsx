import { IconCheck } from "@tabler/icons-react"

import { cn } from "@/lib/utils"

/**
 * The pieces every block of the settings page is made from: a mono caption, the
 * card under it, a quiet line of explanation below, and the green "done" pill.
 * Keeping them in one place is what keeps the blocks looking like one page.
 */
export function SectionKicker({
  className,
  ...props
}: React.ComponentProps<"span">) {
  return (
    <span
      className={cn(
        "text-muted-foreground font-mono text-[10.5px] font-medium tracking-[0.1em] uppercase",
        className,
      )}
      {...props}
    />
  )
}

export const sectionCard = "bg-card border-border rounded-[12px] border"

export const sectionHint = "text-muted-foreground text-xs leading-[1.35]"

export function StatusPill({ children }: { children: React.ReactNode }) {
  return (
    <span className="flex shrink-0 items-center gap-1 rounded-full bg-[color:var(--success-wash)] px-[11px] py-[5px] text-[11.5px] font-medium text-[color:var(--success)]">
      <IconCheck size={12} aria-hidden="true" />
      {children}
    </span>
  )
}
