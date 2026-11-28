import { Suspense } from "react"
import { PathScreen } from "@/features/path"

export default function PathPage() {
  return (
    <Suspense>
      <PathScreen />
    </Suspense>
  )
}
