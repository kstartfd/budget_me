import { Skeleton } from "@/components/ui/skeleton"

export function PlansSkeleton() {
  return (
    <div aria-hidden className="flex flex-col gap-4 lg:gap-[26px]">
      <div className="flex items-end justify-between gap-3">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-7 w-32 rounded-lg" />
          <Skeleton className="h-5 w-40 rounded" />
        </div>
        <Skeleton className="hidden h-11 w-40 rounded-xl lg:block" />
      </div>
      <Skeleton className="h-11 w-full rounded-xl lg:hidden" />
      <div className="flex flex-col gap-2 lg:gap-0">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} className="h-[62px] w-full rounded-[14px]" />
        ))}
      </div>
    </div>
  )
}
