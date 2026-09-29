import { LearningScreen } from "@/features/learning/components/learning-screen"

export default async function TopicPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  return <LearningScreen key={id} topicId={id} />
}
