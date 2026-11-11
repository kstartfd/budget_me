import { RulePage } from "@/features/path"

export default async function RuleRoute({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  return <RulePage key={id} id={id} />
}
