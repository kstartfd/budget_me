/** Phone-only action strip pinned to the bottom edge, above the home indicator. */
export function BottomBar({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-background fixed inset-x-0 bottom-0 z-20 lg:hidden">
      <div className="mx-auto flex max-w-md flex-col gap-2 px-4 pt-2.5 pb-[max(env(safe-area-inset-bottom),1.25rem)]">
        {children}
      </div>
    </div>
  )
}
