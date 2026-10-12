import { OverviewSkeleton } from "./skeleton"

// Shown the instant a tab is tapped, while the route's RSC payload is still
// in flight — without it Next holds the previous screen on the display and
// the app reads as frozen. The screen's own skeleton, so nothing shifts when
// the data lands.
export default function Loading() {
  return <OverviewSkeleton />
}
