import { Skeleton } from "@/components/ui/skeleton"

export function LimitsSkeleton() {
  return (
    <div aria-hidden className="flex flex-col gap-4">
      {/* title + summary */}
      <div className="flex items-baseline justify-between px-0.5">
        <Skeleton className="h-6 w-32 rounded-lg" />
        <Skeleton className="h-4 w-24 rounded" />
      </div>

      {/* new limit button */}
      <Skeleton className="h-[52px] rounded-[18px]" />

      {/* limit cards */}
      <div className="flex flex-col gap-3">
        {Array.from({ length: 4 }, (_, i) => (
          <div
            key={i}
            className="bg-card border-border rounded-[20px] border px-[18px] py-[17px] shadow-[var(--shadow-card)]"
          >
            <div className="flex items-center gap-[11px]">
              <Skeleton className="size-[11px] shrink-0 rounded-full" />
              <Skeleton className="h-4 flex-1 rounded" />
              <Skeleton className="h-5 w-12 rounded-full" />
            </div>
            <Skeleton className="mt-[13px] h-[9px] w-full rounded-full" />
            <div className="mt-[9px] flex items-baseline justify-between">
              <Skeleton className="h-4 w-28 rounded" />
              <Skeleton className="h-3.5 w-16 rounded" />
            </div>
            <div className="border-rowline mt-[13px] flex gap-[9px] border-t pt-3">
              <Skeleton className="h-[38px] flex-1 rounded-xl" />
              <Skeleton className="h-[38px] w-24 rounded-xl" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
