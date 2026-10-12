import { Skeleton } from "@/components/ui/skeleton"

const card =
  "bg-card border-border rounded-[24px] border shadow-[var(--shadow-card)]"

export function OverviewSkeleton() {
  return (
    <div
      aria-hidden
      className="flex flex-col gap-4 lg:grid lg:grid-cols-2 lg:gap-5"
    >
      {/* hero */}
      <section className={`${card} px-5 py-[22px] lg:col-span-2 lg:px-7 lg:py-7`}>
        <div className="flex items-center justify-between">
          <Skeleton className="h-3 w-24 rounded" />
          <Skeleton className="h-6 w-20 rounded-full" />
        </div>
        <Skeleton className="mt-3 h-9 w-44 rounded-lg" />
        <Skeleton className="mt-2 h-4 w-52 rounded" />
        <Skeleton className="mt-4 h-2 w-full rounded-full" />
        <div className="mt-4 grid grid-cols-2 gap-2">
          {Array.from({ length: 2 }, (_, i) => (
            <div key={i}>
              <Skeleton className="h-2.5 w-12 rounded" />
              <Skeleton className="mt-2 h-4 w-16 rounded" />
            </div>
          ))}
        </div>
      </section>

      {/* AI teaser */}
      <Skeleton className="h-[104px] rounded-[22px] lg:col-span-2" />

      {/* by category */}
      <section className={`${card} p-[18px]`}>
        <div className="flex items-center justify-between pb-3">
          <Skeleton className="h-4 w-28 rounded" />
          <Skeleton className="h-3 w-14 rounded" />
        </div>
        <div className="flex flex-col gap-3">
          {Array.from({ length: 4 }, (_, i) => (
            <div key={i} className="flex items-center gap-3">
              <Skeleton className="size-2.5 shrink-0 rounded-full" />
              <Skeleton className="h-4 flex-1 rounded" />
              <Skeleton className="h-1.5 w-16 rounded-full" />
              <Skeleton className="h-4 w-[70px] rounded" />
            </div>
          ))}
        </div>
      </section>

      {/* recent */}
      <section className={`${card} overflow-hidden`}>
        <div className="flex items-center justify-between px-4 pt-4 pb-1">
          <Skeleton className="h-4 w-24 rounded" />
          <Skeleton className="h-3 w-14 rounded" />
        </div>
        <div className="divide-rowline divide-y">
          {Array.from({ length: 3 }, (_, i) => (
            <div key={i} className="flex items-center gap-3 px-4 py-3.5">
              <Skeleton className="size-9 shrink-0 rounded-[11px]" />
              <div className="min-w-0 flex-1">
                <Skeleton className="h-4 w-32 rounded" />
                <Skeleton className="mt-1.5 h-3 w-20 rounded" />
              </div>
              <Skeleton className="h-4 w-16 rounded" />
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
