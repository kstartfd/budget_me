import { Skeleton } from "@/components/ui/skeleton"

export function PlansSkeleton() {
  return (
    <div aria-hidden className="flex flex-col gap-4">
      <div className="flex items-baseline justify-between px-0.5">
        <Skeleton className="h-6 w-32 rounded-lg" />
        <Skeleton className="h-4 w-24 rounded" />
      </div>
      <Skeleton className="h-[52px] w-full rounded-[18px]" />
      <div className="flex flex-col gap-2.5">
        {Array.from({ length: 3 }, (_, i) => (
          <Skeleton key={i} className="h-[62px] w-full rounded-[18px]" />
        ))}
      </div>
    </div>
  )
}
