import { cn } from "@/lib/utils"

/** The white block every part of the overview sits in. */
export const card = "bg-card border-border rounded-xl border"

/** The small uppercase caption above a heading. */
export function Caption({
  className,
  ...props
}: React.ComponentProps<"span">) {
  return (
    <span
      className={cn(
        "text-muted-foreground text-xs font-medium tracking-[0.04em] uppercase",
        className,
      )}
      {...props}
    />
  )
}

const tones = {
  neutral: "bg-muted text-foreground",
  muted: "bg-muted text-muted-foreground",
  learning: "bg-info/10 text-info",
  mastery: "bg-success/10 text-success",
  review: "bg-warn/10 text-warn",
} as const

export type Tone = keyof typeof tones

/** A rounded status pill: the colour says what kind of status it is. */
export function Pill({
  tone = "neutral",
  className,
  ...props
}: React.ComponentProps<"span"> & { tone?: Tone }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap",
        tones[tone],
        className,
      )}
      {...props}
    />
  )
}

/** A big tappable choice that holds a title and a line under it. */
export function Choice({
  selected,
  className,
  ...props
}: React.ComponentProps<"button"> & { selected?: boolean }) {
  return (
    <button
      type="button"
      className={cn(
        "border-border bg-card focus-visible:ring-ring/60 w-full cursor-pointer rounded-[10px] border px-3.5 py-3 text-left text-sm transition-colors outline-none focus-visible:ring-2 disabled:cursor-not-allowed disabled:opacity-50",
        selected
          ? "border-foreground bg-muted ring-foreground ring-1"
          : "hover:bg-muted/60",
        className,
      )}
      {...props}
    />
  )
}

/** A rounded toggle chip for a multiple choice. */
export function Chip({
  pressed,
  className,
  ...props
}: React.ComponentProps<"button"> & { pressed: boolean }) {
  return (
    <button
      type="button"
      aria-pressed={pressed}
      className={cn(
        "border-border bg-card focus-visible:ring-ring/60 min-h-10 cursor-pointer rounded-full border px-3.5 text-sm transition-colors outline-none focus-visible:ring-2",
        pressed
          ? "border-foreground bg-muted ring-foreground ring-1"
          : "hover:bg-muted/60",
        className,
      )}
      {...props}
    />
  )
}

/** A row of options of which one is on: the muted track with the pick raised. */
export function Segmented<T extends string | number>({
  label,
  options,
  value,
  onChange,
  format,
}: {
  label: string
  options: readonly T[]
  value: T
  onChange: (value: T) => void
  format: (value: T) => string
}) {
  return (
    <div
      role="radiogroup"
      aria-label={label}
      className="bg-muted flex gap-1 rounded-lg p-[3px]"
    >
      {options.map((option) => {
        const on = option === value

        return (
          <button
            key={option}
            type="button"
            role="radio"
            aria-checked={on}
            onClick={() => onChange(option)}
            className={cn(
              "focus-visible:ring-ring/60 h-10 min-w-14 flex-1 cursor-pointer rounded-md px-3 text-[13px] transition-colors outline-none focus-visible:ring-2",
              on
                ? "bg-card text-foreground font-semibold shadow-sm"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            {format(option)}
          </button>
        )
      })}
    </div>
  )
}

/** The two-letter badge of a language: its code on a muted square. */
export function CodeBadge({
  code,
  className,
}: {
  code: string
  className?: string
}) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "bg-muted text-foreground flex size-6 shrink-0 items-center justify-center rounded-md text-[11px] font-semibold uppercase",
        className,
      )}
    >
      {code.slice(0, 2)}
    </span>
  )
}
