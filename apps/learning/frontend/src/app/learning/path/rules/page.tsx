import { RulesIndex } from "@/features/path"

export default async function RulesPage({
  searchParams,
}: {
  searchParams: Promise<{ language?: string }>
}) {
  const { language } = await searchParams

  return <RulesIndex key={language} language={language} />
}
