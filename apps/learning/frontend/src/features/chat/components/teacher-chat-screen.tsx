"use client"

import { useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import {
  IconEraser,
  IconHistory,
  IconSchool,
  IconSettings,
} from "@tabler/icons-react"

import { useClaimFullScreen } from "@/components/full-screen"
import { ScreenBar } from "@/components/screen-bar"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { useLearning } from "@/features/learning/hooks"
import { useLanguageName } from "@/features/learning/language"
import { PathScreen } from "@/features/path"
import { cn } from "@/lib/utils"

import {
  useArchiveTopicChat,
  useTopicChat,
  useTopicChatTurn,
} from "../hooks"
import {
  Bubble,
  chatLimits,
  Composer,
  LimitReached,
  useChatFailure,
} from "./chat-ui"
import { DialogHistory } from "./dialog-history"

const starters = ["beginner", "talk", "check", "grammar"] as const

const lastTopicKey = "learning:chat-topic"

function rememberedTopic(): string | null {
  try {
    return window.localStorage.getItem(lastTopicKey)
  } catch {
    return null
  }
}

function remember(id: string) {
  try {
    window.localStorage.setItem(lastTopicKey, id)
  } catch {
    // Private mode and the like: the choice just is not kept.
  }
}

const circleButton =
  "bg-secondary border-border text-muted-foreground size-10 shrink-0 rounded-xl border"

/**
 * A free conversation with an AI teacher of the topic's language, about no
 * saved word. Clearing it keeps the dialog in the history.
 */
export function TeacherChatScreen() {
  const t = useTranslations("chat")
  const learning = useTranslations("learning")
  const languageName = useLanguageName()
  const { topics } = useLearning()
  // Read once. Nothing that depends on it is drawn before the topics have
  // loaded, so the server and the first client render still agree.
  const [picked, setPicked] = useState<string | null>(rememberedTopic)
  useClaimFullScreen()

  const topic =
    topics.data?.find((item) => item.id === picked) ?? topics.data?.[0]
  // A level test owns the screen while it runs: the conversation waits out.
  const [programBusy, setProgramBusy] = useState(false)

  return (
    <div className="anim-fade -mb-44 flex flex-1 flex-col gap-3 lg:mx-auto lg:-mb-16 lg:w-full lg:max-w-[760px] lg:pt-3">
      <ScreenBar
        className="lg:hidden"
        backLabel={learning("title")}
        backHref="/learning"
        title={t("teacherTitle")}
      />
      <header className="hidden lg:block">
        <h1 className="text-[28px] font-bold tracking-[-0.02em]">
          {t("teacherTitle")}
        </h1>
        <p className="text-muted-foreground mt-1 text-[13px]">
          {t("teacherSubtitle")}
        </p>
      </header>

      {topics.isPending ? (
        <Skeleton className="h-40 w-full rounded-[18px]" />
      ) : topics.isError ? (
        <p role="alert" className="text-muted-foreground text-sm">
          {learning("loadError")}
        </p>
      ) : !topic ? (
        <div className="bg-card border-border rounded-[18px] border p-6">
          <p className="text-sm font-semibold">{t("noTopics")}</p>
          <p className="text-muted-foreground mt-2 text-sm leading-relaxed">
            {t("noTopicsHint")}
          </p>
          <Button asChild variant="secondary" className="text-primary mt-4">
            <Link href="/learning">{learning("title")}</Link>
          </Button>
        </div>
      ) : (
        <>
          <PathScreen
            language={topic.language}
            translationLanguage={topic.translationLanguage}
            embedded
            onTestChange={setProgramBusy}
          />
          {!programBusy && (
            <Conversation
              key={topic.id}
              topicId={topic.id}
              picker={
                <Select
                  value={topic.id}
                  onValueChange={(id) => {
                    setPicked(id)
                    remember(id)
                  }}
                >
                  <SelectTrigger
                    aria-label={t("topic")}
                    className="bg-card border-border h-10 min-w-0 flex-1 rounded-xl px-3.5 text-sm"
                  >
                    <SelectValue>
                      <span className="truncate">
                        {topic.name}
                        <span className="text-faint">
                          {" · "}
                          {languageName(topic.language)}
                        </span>
                      </span>
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {topics.data.map((item) => (
                      <SelectItem key={item.id} value={item.id}>
                        {item.name} · {languageName(item.language)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              }
            />
          )}
        </>
      )}
    </div>
  )
}

function Conversation({
  topicId,
  picker,
}: {
  topicId: string
  picker: React.ReactNode
}) {
  const t = useTranslations("chat")
  const failure = useChatFailure()
  const chat = useTopicChat(topicId)
  const { turn, error, send, sending } = useTopicChatTurn(topicId)
  const archive = useArchiveTopicChat(topicId)
  const [text, setText] = useState("")
  const [confirmClear, setConfirmClear] = useState(false)
  const [history, setHistory] = useState(false)
  const end = useRef<HTMLDivElement>(null)

  const messages = chat.data?.messages ?? []
  const streaming = turn?.answer.length ?? 0
  const { exhausted, usedPercent } = chatLimits(chat.data)

  useEffect(() => {
    end.current?.scrollIntoView({ block: "end" })
  }, [messages.length, streaming, turn?.question])

  /** Sends what is in the field, or a starter's message and leaves the field alone. */
  const submit = async (starter?: string) => {
    const question = (starter ?? text).trim()
    if (!question || sending) return

    // The question is on screen as a bubble the moment it is sent, so it leaves
    // the field; a failed turn stores nothing and gives the text back, unless
    // the learner has already started something else.
    if (starter === undefined) setText("")

    if (!(await send(question)) && starter === undefined)
      setText((current) => current || question)
  }

  const toolbar = (
    <div className="flex items-center gap-2">
      {picker}
      <Button
        variant="secondary"
        size="icon"
        aria-label={t("history")}
        className={circleButton}
        onClick={() => setHistory(true)}
      >
        <IconHistory size={17} />
      </Button>
      <Button
        variant="secondary"
        size="icon"
        aria-label={t("clear")}
        disabled={sending || messages.length === 0}
        className={circleButton}
        onClick={() => setConfirmClear(true)}
      >
        <IconEraser size={17} />
      </Button>
    </div>
  )

  return (
    <>
      {toolbar}

      {confirmClear && (
        <div className="bg-card border-border space-y-3 rounded-[14px] border p-3.5">
          <p className="text-sm leading-relaxed">{t("clearDialogConfirm")}</p>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={archive.isPending}
              onClick={() => setConfirmClear(false)}
            >
              {t("cancel")}
            </Button>
            <Button
              size="sm"
              disabled={archive.isPending}
              onClick={() =>
                archive.mutate(undefined, {
                  onSuccess: () => setConfirmClear(false),
                })
              }
            >
              {t("clear")}
            </Button>
          </div>
          {archive.isError && (
            <p role="alert" className="text-destructive text-sm">
              {t("clearFailed")}
            </p>
          )}
        </div>
      )}

      {chat.isError ? (
        <p role="alert" className="text-muted-foreground text-sm">
          {t("loadFailed")}
        </p>
      ) : chat.isPending ? (
        <Skeleton className="h-40 w-full rounded-[18px]" />
      ) : !chat.data.available ? (
        <div className="bg-card border-border rounded-[18px] border p-6">
          <p className="text-sm font-semibold">{t("needAgent")}</p>
          <p className="text-muted-foreground mt-2 text-sm leading-relaxed">
            {t("needAgentHint")}
          </p>
          <Button asChild variant="secondary" className="text-primary mt-4">
            <Link href="/learning/settings">
              <IconSettings size={16} />
              {t("openSettings")}
            </Link>
          </Button>
        </div>
      ) : (
        <>
          <div className="flex flex-1 flex-col">
            {messages.length === 0 && !turn ? (
              <div className="m-auto w-full space-y-5 py-6 text-center">
                <IconSchool size={34} className="text-primary mx-auto" />
                <div className="space-y-1.5">
                  <p className="text-base font-semibold">{t("welcome")}</p>
                  <p className="text-muted-foreground mx-auto max-w-[300px] text-[13px] leading-[1.5]">
                    {t("welcomeHint")}
                  </p>
                </div>
                <div className="grid gap-2 text-left sm:grid-cols-2">
                  {starters.map((id) => (
                    <Button
                      key={id}
                      variant="secondary"
                      disabled={sending || exhausted}
                      className="h-auto justify-start rounded-2xl px-4 py-3 text-left text-[13px] font-medium whitespace-normal"
                      onClick={() => void submit(t(`starters.${id}.message`))}
                    >
                      {t(`starters.${id}.label`)}
                    </Button>
                  ))}
                </div>
              </div>
            ) : (
              <div className="space-y-2" aria-live="polite">
                {messages.map((m) => (
                  <Bubble key={m.id} role={m.role}>
                    {m.content}
                  </Bubble>
                ))}
                {turn && (
                  <>
                    <Bubble role="user">{turn.question}</Bubble>
                    <Bubble role="assistant">
                      {turn.answer || t("typing")}
                    </Bubble>
                  </>
                )}
              </div>
            )}
            <div ref={end} className="scroll-mb-44" />
          </div>

          <div
            className={cn(
              "bg-background sticky bottom-0 z-10 -mx-[18px] space-y-2 px-[18px] pt-2",
              "pb-[max(env(safe-area-inset-bottom),1rem)] lg:mx-0 lg:px-0 lg:pb-6",
            )}
          >
            {error && (
              <p role="alert" className="text-destructive text-sm">
                {failure(error)}
              </p>
            )}
            {exhausted ? (
              <LimitReached chat={chat.data} />
            ) : (
              <Composer
                value={text}
                onChange={setText}
                onSubmit={() => void submit()}
                sending={sending}
                placeholder={t("placeholderTeacher")}
              />
            )}
            {!exhausted && usedPercent >= 70 && (
              <p className="text-faint text-xs">
                {t("budgetLow", { percent: usedPercent })}
              </p>
            )}
          </div>
        </>
      )}

      {history && (
        <DialogHistory topicId={topicId} onClose={() => setHistory(false)} />
      )}
    </>
  )
}
