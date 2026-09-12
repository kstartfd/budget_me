import { DaySkeleton } from "./skeleton"

// Shown while the route's RSC payload is in flight, so tapping the tab never
// leaves the previous screen frozen on the display.
export default function Loading() {
  return <DaySkeleton />
}
