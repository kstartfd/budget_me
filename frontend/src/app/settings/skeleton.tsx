import { Skeleton } from "@/components/ui/skeleton"

export function SettingsSkeleton() {
  return (
    <div aria-hidden className="flex flex-col gap-[18px]">
      <Skeleton className="h-6 w-40 rounded-lg" />

      {/* your id */}
      <div className="bg-card border-border flex items-center gap-3 rounded-[18px] border p-[14px] shadow-[var(--shadow-card)]">
        <div className="min-w-0 flex-1">
          <Skeleton className="h-3.5 w-20 rounded" />
          <Skeleton className="mt-2 h-3 w-56 rounded" />
        </div>
        <Skeleton className="size-[34px] shrink-0 rounded-full" />
      </div>

      {/* billing period */}
      <section className="flex flex-col gap-[9px]">
        <Skeleton className="ml-0.5 h-3 w-32 rounded" />
        <div className="bg-card border-border rounded-[18px] border px-[15px] pt-[15px] pb-[13px] shadow-[var(--shadow-card)]">
          <Skeleton className="h-4 w-40 rounded" />
          <Skeleton className="mt-2 h-3 w-56 rounded" />
          <Skeleton className="mt-3 h-16 w-full rounded-xl" />
          <Skeleton className="mx-auto mt-2 h-3 w-40 rounded" />
          <Skeleton className="mt-3 h-10 w-full rounded-xl" />
        </div>
      </section>

      {/* main currency */}
      <section className="flex flex-col gap-[9px]">
        <Skeleton className="ml-0.5 h-3 w-24 rounded" />
        <div className="bg-card border-border rounded-[18px] border p-[14px] shadow-[var(--shadow-card)]">
          <div className="flex items-center gap-3">
            <div className="flex-1">
              <Skeleton className="h-3.5 w-36 rounded" />
              <Skeleton className="mt-2 h-3 w-48 rounded" />
            </div>
            <Skeleton className="h-9 w-20 rounded-xl" />
          </div>
        </div>
      </section>

      {/* categories link */}
      <section className="flex flex-col gap-[9px]">
        <Skeleton className="ml-0.5 h-3 w-16 rounded" />
        <div className="bg-card border-border flex items-center gap-3 rounded-[18px] border p-[14px] shadow-[var(--shadow-card)]">
          <Skeleton className="size-[38px] shrink-0 rounded-xl" />
          <div className="min-w-0 flex-1">
            <Skeleton className="h-4 w-28 rounded" />
            <Skeleton className="mt-1.5 h-3 w-40 rounded" />
          </div>
          <Skeleton className="size-4 shrink-0 rounded" />
        </div>
      </section>
    </div>
  )
}
