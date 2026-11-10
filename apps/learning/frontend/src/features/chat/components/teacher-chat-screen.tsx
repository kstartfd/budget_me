"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import {
  IconBook2,
  IconHistory,
  IconListCheck,
  IconMessages,
  IconPlus,
  IconSchool,
  IconSeedling,
  IconSettings,
  IconWorld,
} from "@tabler/icons-react";

import { useClaimFullScreen } from "@/components/full-screen";
import { ScreenBar } from "@/components/screen-bar";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { useLearning } from "@/features/learning/hooks";
import { useLanguageName } from "@/features/learning/language";
import { cn } from "@/lib/utils";

import { useArchiveTopicChat, useTopicChat, useTopicChatTurn } from "../hooks";
import {
  Bubble,
  chatLimits,
  Composer,
  LimitReached,
  useChatFailure,
} from "./chat-ui";
import { DialogHistory } from "./dialog-history";

const starters = [
  { id: "beginner", icon: IconSeedling },
  { id: "talk", icon: IconMessages },
  { id: "check", icon: IconListCheck },
  { id: "grammar", icon: IconBook2 },
] as const;

const quickReplies = ["explain", "hint"] as const;

const lastTopicKey = "learning:chat-topic";

function rememberedTopic(): string | null {
  try {
    return window.localStorage.getItem(lastTopicKey);
  } catch {
    return null;
  }
}

function remember(id: string) {
  try {
    window.localStorage.setItem(lastTopicKey, id);
  } catch {
    // Private mode and the like: the choice just is not kept.
  }
}

const iconButton =
  "border-border bg-background text-muted-foreground hover:bg-muted hover:text-foreground size-9 shrink-0 rounded-md border";

/**
 * A free conversation with an AI teacher of the topic's language, about no
 * saved word. Clearing it keeps the dialog in the history.
 */
export function TeacherChatScreen() {
  const t = useTranslations("chat");
  const learning = useTranslations("learning");
  const languageName = useLanguageName();
  const { topics } = useLearning();
  // A guided exercise links here with the notebook and the question to ask.
  // window.location is still the previous page while a client-side navigation
  // renders, so these come from the router, not from the address bar.
  const params = useSearchParams();
  // Read once. Nothing that depends on it is drawn before the topics have
  // loaded, so the server and the first client render still agree.
  const [picked, setPicked] = useState<string | null>(
    () => params.get("topic") ?? rememberedTopic(),
  );
  const [draftQuestion, setDraftQuestion] = useState(
    () => params.get("question") ?? "",
  );
  useClaimFullScreen();

  const topic =
    topics.data?.find((item) => item.id === picked) ?? topics.data?.[0];

  const heading = (
    <header className="hidden min-w-0 flex-1 lg:block">
      <h1 className="text-2xl font-semibold tracking-[-0.01em]">
        {t("teacherTitle")}
      </h1>
      <p className="text-muted-foreground mt-1 max-w-[280px] text-sm">
        {t("teacherSubtitle")}
      </p>
    </header>
  );

  return (
    <div className="anim-fade -mb-44 flex flex-1 flex-col gap-3 lg:-mb-16 lg:mx-auto lg:w-full lg:max-w-[720px] lg:gap-5">
      <ScreenBar
        className="lg:hidden"
        backLabel={learning("title")}
        backHref="/learning"
        title={t("teacherTitle")}
      />

      {topics.isPending ? (
        <>
          {heading}
          <Skeleton className="h-40 w-full rounded-lg" />
        </>
      ) : topics.isError ? (
        <>
          {heading}
          <p role="alert" className="text-muted-foreground text-sm">
            {learning("loadError")}
          </p>
        </>
      ) : !topic ? (
        <>
          {heading}
          <div className="bg-card border-border rounded-lg border p-6">
            <p className="text-sm font-semibold">{t("noTopics")}</p>
            <p className="text-muted-foreground mt-2 text-sm leading-relaxed">
              {t("noTopicsHint")}
            </p>
            <Button asChild variant="outline" size="lg" className="mt-4">
              <Link href="/learning">{learning("title")}</Link>
            </Button>
          </div>
        </>
      ) : (
        <Conversation
          key={topic.id}
          topicId={topic.id}
          heading={heading}
          initialQuestion={draftQuestion}
          onInitialQuestionUsed={() => setDraftQuestion("")}
          picker={
            <Select
              value={topic.id}
              onValueChange={(id) => {
                setPicked(id);
                remember(id);
              }}
            >
              <SelectTrigger
                aria-label={t("topic")}
                className="border-border bg-background h-9 min-w-0 flex-1 justify-start gap-2 rounded-md px-3 text-sm font-medium lg:w-60 lg:flex-none *:data-[slot=select-value]:flex-1 *:data-[slot=select-value]:text-left"
              >
                <IconWorld
                  size={14}
                  className="text-muted-foreground shrink-0"
                />
                <SelectValue>
                  <span className="truncate">
                    {topic.name}
                    <span className="text-faint font-normal">
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
    </div>
  );
}

function Conversation({
  topicId,
  heading,
  initialQuestion,
  onInitialQuestionUsed,
  picker,
}: {
  topicId: string;
  heading: React.ReactNode;
  initialQuestion: string;
  onInitialQuestionUsed: () => void;
  picker: React.ReactNode;
}) {
  const t = useTranslations("chat");
  const failure = useChatFailure();
  const chat = useTopicChat(topicId);
  const { turn, error, send, sending } = useTopicChatTurn(topicId);
  const archive = useArchiveTopicChat(topicId);
  const [text, setText] = useState(initialQuestion);
  const [confirmClear, setConfirmClear] = useState(false);
  const [history, setHistory] = useState(false);
  const end = useRef<HTMLDivElement>(null);

  // A question brought from a guided exercise belongs in the composer. Consume
  // it once so switching the chat topic does not put it back into the field.
  useEffect(() => {
    if (initialQuestion) onInitialQuestionUsed();
  }, [initialQuestion, onInitialQuestionUsed]);

  const messages = chat.data?.messages ?? [];
  const streaming = turn?.answer.length ?? 0;
  const { exhausted, usedPercent } = chatLimits(chat.data);

  useEffect(() => {
    end.current?.scrollIntoView({ block: "end" });
  }, [messages.length, streaming, turn?.question]);

  /** Sends what is in the field, or a starter's message and leaves the field alone. */
  const submit = async (starter?: string) => {
    const question = (starter ?? text).trim();
    if (!question || sending) return;

    // The question is on screen as a bubble the moment it is sent, so it leaves
    // the field; a failed turn stores nothing and gives the text back, unless
    // the learner has already started something else.
    if (starter === undefined) setText("");

    if (!(await send(question)) && starter === undefined)
      setText((current) => current || question);
  };

  const toolbar = (
    <div className="flex shrink-0 items-center gap-2">
      {picker}
      <Button
        variant="outline"
        size="icon-lg"
        aria-label={t("history")}
        className={iconButton}
        onClick={() => setHistory(true)}
      >
        <IconHistory size={15} />
      </Button>
      <Button
        size="lg"
        aria-label={t("newLesson")}
        disabled={sending || messages.length === 0}
        className="size-9 shrink-0 p-0 lg:h-9 lg:w-auto lg:px-3.5 lg:text-sm"
        onClick={() => setConfirmClear(true)}
      >
        <IconPlus size={14} />
        <span className="hidden lg:inline">{t("newLesson")}</span>
      </Button>
    </div>
  );

  return (
    <>
      <div className="border-border flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between lg:gap-4 lg:border-b lg:pb-5">
        {heading}
        {toolbar}
      </div>

      <div className="flex flex-1 flex-col gap-3 lg:w-full lg:gap-5">
        {confirmClear && (
          <div className="bg-card border-border space-y-3 rounded-lg border p-4">
            <p className="text-sm leading-relaxed">{t("clearDialogConfirm")}</p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="lg"
                disabled={archive.isPending}
                onClick={() => setConfirmClear(false)}
              >
                {t("cancel")}
              </Button>
              <Button
                size="lg"
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
          <Skeleton className="h-40 w-full rounded-lg" />
        ) : !chat.data.available ? (
          <div className="bg-card border-border rounded-lg border p-6">
            <p className="text-sm font-semibold">{t("needAgent")}</p>
            <p className="text-muted-foreground mt-2 text-sm leading-relaxed">
              {t("needAgentHint")}
            </p>
            <Button asChild variant="outline" size="lg" className="mt-4">
              <Link href="/learning/settings">
                <IconSettings size={15} />
                {t("openSettings")}
              </Link>
            </Button>
          </div>
        ) : (
          <>
            <div className="flex flex-1 flex-col">
              {messages.length === 0 && !turn ? (
                <div className="m-auto flex w-full flex-col items-center gap-5 py-6 text-center">
                  <span className="bg-muted flex size-12 items-center justify-center rounded-lg">
                    <IconSchool size={22} />
                  </span>
                  <div className="space-y-2">
                    <p className="text-[22px] font-semibold tracking-[-0.01em] lg:text-3xl">
                      {t("welcome")}
                    </p>
                    <p className="text-muted-foreground mx-auto max-w-[300px] text-[13px] leading-relaxed lg:max-w-[440px] lg:text-[15px]">
                      {t("welcomeHint")}
                    </p>
                  </div>
                  <div className="grid w-full gap-2 text-left lg:max-w-[640px] lg:grid-cols-2 lg:gap-3">
                    {starters.map(({ id, icon: Icon }) => (
                      <button
                        key={id}
                        type="button"
                        disabled={sending || exhausted}
                        className="border-border bg-background hover:bg-muted/50 flex items-center gap-3 rounded-lg border p-3 text-left transition-colors disabled:opacity-50 lg:p-4"
                        onClick={() => void submit(t(`starters.${id}.message`))}
                      >
                        <span className="bg-muted flex size-8 shrink-0 items-center justify-center rounded-md lg:size-9">
                          <Icon size={15} />
                        </span>
                        <span className="min-w-0">
                          <span className="block text-sm font-semibold">
                            {t(`starters.${id}.title`)}
                          </span>
                          <span className="text-muted-foreground block text-[13px]">
                            {t(`starters.${id}.hint`)}
                          </span>
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="space-y-4" aria-live="polite">
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
                "bg-background sticky bottom-0 z-10 -mx-4 space-y-2 px-4 pt-2",
                "pb-[max(env(safe-area-inset-bottom),1rem)] lg:mx-0 lg:px-0 lg:pb-6",
              )}
            >
              {error && (
                <p role="alert" className="text-destructive text-sm">
                  {failure(error)}
                </p>
              )}
              {!exhausted &&
                !turn &&
                !sending &&
                messages.at(-1)?.role === "assistant" && (
                  <div className="flex flex-wrap gap-2">
                    {quickReplies.map((id) => (
                      <button
                        key={id}
                        type="button"
                        className="border-input bg-background hover:bg-muted rounded-full border px-3 py-1.5 text-xs font-medium transition-colors"
                        onClick={() => void submit(t(`quick.${id}`))}
                      >
                        {t(`quick.${id}`)}
                      </button>
                    ))}
                  </div>
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
      </div>

      {history && (
        <DialogHistory topicId={topicId} onClose={() => setHistory(false)} />
      )}
    </>
  );
}
