import Link from "next/link"
import { IconChevronLeft } from "@tabler/icons-react"

/**
 * Phone top bar of a nested screen: ‹ back on the left, the title in the
 * middle, one round action on the right.
 */
export function ScreenBar({
  backLabel,
  backHref,
  onBack,
  title,
  action,
  className = "",
}: {
  backLabel: string
  backHref?: string
  onBack?: () => void
  title: string
  action?: React.ReactNode
  className?: string
}) {
  const back = (
    <>
      <IconChevronLeft size={18} className="-ml-1 shrink-0" />
      <span className="truncate">{backLabel}</span>
    </>
  )
  const backClass =
    "text-primary flex min-w-0 items-center justify-self-start text-[15px]"

  return (
    <div
      className={`grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2 ${className}`}
    >
      {backHref ? (
        <Link href={backHref} className={backClass}>
          {back}
        </Link>
      ) : (
        <button type="button" onClick={onBack} className={backClass}>
          {back}
        </button>
      )}
      <h1 className="max-w-[46vw] truncate text-[15px] font-semibold">
        {title}
      </h1>
      <div className="flex justify-self-end">{action}</div>
    </div>
  )
}
