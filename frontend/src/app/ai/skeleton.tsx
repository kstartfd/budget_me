import { Skeleton } from "@/components/ui/skeleton"

export function AiSkeleton() {
  return (
    <div aria-hidden className="flex flex-col gap-4">
      <Skeleton className="h-6 w-32 rounded-lg" />
      <Skeleton className="h-28 w-full rounded-[22px]" />
      <Skeleton className="h-24 w-full rounded-[22px]" />
      {Array.from({ length: 2 }, (_, i) => (
        <Skeleton key={i} className="h-[92px] w-full rounded-[20px]" />
      ))}
    </div>
  )
}
