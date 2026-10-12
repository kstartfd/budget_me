import { Skeleton } from "@/components/ui/skeleton"

const card =
  "bg-card border-border rounded-[22px] border shadow-[var(--shadow-card)]"

export function AnalyticsSkeleton() {
  return (
    <div aria-hidden className="flex flex-col gap-4">
      <Skeleton className="h-6 w-40 rounded-lg" />

      {/* donut + legend */}
      <section className={`${card} px-5 py-[22px]`}>
        <div className="flex items-center gap-[22px]">
          <Skeleton className="size-[130px] shrink-0 rounded-full" />
          <div className="flex flex-1 flex-col gap-[11px]">
            {Array.from({ length: 4 }, (_, i) => (
              <div key={i} className="flex items-center gap-[9px]">
                <Skeleton className="size-2 shrink-0 rounded-full" />
                <Skeleton className="h-3.5 flex-1 rounded" />
                <Skeleton className="h-3.5 w-8 rounded" />
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* trend bars */}
      <section className={`${card} p-5`}>
        <div className="flex items-baseline justify-between">
          <Skeleton className="h-4 w-28 rounded" />
          <Skeleton className="h-3 w-16 rounded" />
        </div>
        <div className="mt-5 flex h-[118px] items-end gap-2.5">
          {[40, 62, 48, 80, 55, 95].map((h, i) => (
            <div
              key={i}
              className="flex h-full flex-1 flex-col items-center justify-end gap-2"
            >
              <Skeleton
                className="w-full rounded-t-[7px] rounded-b-[4px]"
                style={{ height: `${h}%` }}
              />
              <Skeleton className="h-2.5 w-6 rounded" />
            </div>
          ))}
        </div>
      </section>

      {/* where money goes */}
      <section>
        <Skeleton className="mb-3 ml-1 h-4 w-36 rounded" />
        <div className="bg-card border-border divide-rowline divide-y overflow-hidden rounded-[20px] border shadow-[var(--shadow-card)]">
          {Array.from({ length: 5 }, (_, i) => (
            <div key={i} className="flex items-center gap-[13px] px-4 py-3.5">
              <Skeleton className="size-2.5 shrink-0 rounded-full" />
              <Skeleton className="h-4 flex-1 rounded" />
              <Skeleton className="h-3.5 w-10 rounded" />
              <Skeleton className="h-4 w-16 rounded" />
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
