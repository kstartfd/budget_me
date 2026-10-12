import { ListDetailScreen } from "@/features/lists"

/** `/lists/[id]` — one list's items. */
export default async function ListDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params

  return <ListDetailScreen listId={id} />
}
