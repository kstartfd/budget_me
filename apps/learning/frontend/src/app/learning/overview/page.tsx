import { Suspense } from "react"
import { OverviewScreen } from "@/features/home"

export default function OverviewPage() {
  return (
    <Suspense>
      <OverviewScreen />
    </Suspense>
  )
}
