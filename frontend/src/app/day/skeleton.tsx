import { Skeleton } from "@/components/ui/skeleton"

export function DaySkeleton() {
  return (
    <div aria-hidden className="flex flex-col gap-4 lg:max-w-2xl">
      <div className="bg-card border-border rounded-[22px] border px-5 py-[18px] shadow-[var(--shadow-card)]">
        <div className="flex items-center gap-2">
          <Skeleton className="size-[34px] shrink-0 rounded-[11px]" />
          <div className="flex flex-1 flex-col items-center gap-1.5">
            <Skeleton className="h-3 w-16 rounded" />
            <Skeleton className="h-5 w-36 rounded" />
          </div>
          <Skeleton className="size-[34px] shrink-0 rounded-[11px]" />
        </div>
        <div className="mt-4 flex items-center gap-3">
          <Skeleton className="h-1.5 flex-1 rounded-full" />
          <Skeleton className="h-3 w-12 rounded" />
        </div>
      </div>

      <div className="bg-card border-border divide-rowline divide-y overflow-hidden rounded-[22px] border">
        {Array.from({ length: 4 }, (_, i) => (
          <div key={i} className="flex items-center gap-1 px-2 py-0.5">
            <div className="flex size-11 shrink-0 items-center justify-center">
              <Skeleton className="size-[23px] rounded-full" />
            </div>
            <div className="flex-1 py-2.5">
              <Skeleton className="h-4 w-40 rounded" />
            </div>
            <Skeleton className="size-[30px] shrink-0 rounded-full" />
          </div>
        ))}
      </div>
    </div>
  )
}
