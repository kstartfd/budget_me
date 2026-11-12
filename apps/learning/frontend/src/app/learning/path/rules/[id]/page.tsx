import { RulePage } from "@/features/path"

export default async function RuleRoute({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>
  searchParams: Promise<{ language?: string }>
}) {
  const { id } = await params
  const { language } = await searchParams

  return <RulePage key={`${language}/${id}`} id={id} language={language} />
}
