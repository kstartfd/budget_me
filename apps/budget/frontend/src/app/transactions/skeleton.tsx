import { Skeleton } from "@/components/ui/skeleton"

export function ListSkeleton() {
  return (
    <div aria-hidden className="flex flex-col gap-4">
      {/* title + count */}
      <div className="flex items-baseline justify-between px-0.5">
        <Skeleton className="h-6 w-40 rounded-lg" />
        <Skeleton className="h-4 w-16 rounded" />
      </div>

      {/* filter button */}
      <div className="flex items-center gap-2.5 px-0.5">
        <Skeleton className="h-8 w-24 rounded-full" />
        <Skeleton className="h-3.5 w-28 rounded" />
      </div>

      {/* two day groups */}
      {Array.from({ length: 2 }, (_, g) => (
        <section key={g}>
          <div className="flex items-center justify-between px-1 pb-[9px]">
            <Skeleton className="h-3 w-20 rounded" />
            <Skeleton className="h-3 w-16 rounded" />
          </div>
          <div className="bg-card border-border divide-rowline divide-y overflow-hidden rounded-[20px] border shadow-[var(--shadow-card)]">
            {Array.from({ length: 3 }, (_, i) => (
              <div key={i} className="flex items-center gap-3 px-4 py-3">
                <Skeleton className="size-9 shrink-0 rounded-[11px]" />
                <div className="min-w-0 flex-1">
                  <Skeleton className="h-4 w-32 rounded" />
                  <Skeleton className="mt-1.5 h-3 w-24 rounded" />
                </div>
                <div className="flex flex-col items-end gap-1.5">
                  <Skeleton className="h-4 w-16 rounded" />
                  <Skeleton className="h-3 w-12 rounded" />
                </div>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}
