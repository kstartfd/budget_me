"use client"

import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useLearning } from "../hooks"
import type { Topic } from "../schemas"
import { LearningModal } from "./learning-modal"
import { TopicForm } from "./topic-form"
import { TopicScreen } from "./topic-screen"
import { TopicsList } from "./topics-list"

export function LearningScreen({ topicId = "" }: { topicId?: string }) {
  const t = useTranslations("learning")
  const router = useRouter()
  const { topics, deleteTopic } = useLearning()
  const [topicForm, setTopicForm] = useState<Topic | "new" | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const topic = topics.data?.find((item) => item.id === topicId)

  if (topics.isPending)
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-36" />
        <Skeleton className="h-40 w-full rounded-[22px]" />
      </div>
    )
  if (topics.isError)
    return (
      <p
        role="alert"
        className="text-muted-foreground py-10 text-center text-sm"
      >
        {t("loadError")}
      </p>
    )
  if (topicId && !topic)
    return (
      <div className="space-y-3">
        <p>{t("topicMissing")}</p>
        <Link href="/learning" className="text-primary text-sm">
          {t("back")}
        </Link>
      </div>
    )

  return (
    <>
      {topic ? (
        <TopicScreen
          topic={topic}
          onEditTopic={() => setTopicForm(topic)}
          onDeleteTopic={() => setConfirmDelete(true)}
        />
      ) : (
        <TopicsList
          topics={topics.data ?? []}
          onNewTopic={() => setTopicForm("new")}
        />
      )}
      {topicForm && (
        <TopicForm
          topic={topicForm === "new" ? undefined : topicForm}
          onClose={() => setTopicForm(null)}
        />
      )}
      {confirmDelete && topic && (
        <LearningModal
          title={t("deleteTopic")}
          description={t("deleteTopicWarning")}
          onClose={() => setConfirmDelete(false)}
        >
          <div className="flex gap-3">
            <Button
              variant="secondary"
              className="flex-1"
              disabled={deleteTopic.isPending}
              onClick={() => setConfirmDelete(false)}
            >
              {t("cancel")}
            </Button>
            <Button
              variant="destructive"
              className="flex-1"
              disabled={deleteTopic.isPending}
              onClick={() =>
                deleteTopic.mutate(topic.id, {
                  onSuccess: () => router.push("/learning"),
                })
              }
            >
              {t("delete")}
            </Button>
          </div>
          {deleteTopic.isError && (
            <p role="alert" className="text-destructive text-sm">
              {t("deleteError")}
            </p>
          )}
        </LearningModal>
      )}
    </>
  )
}
