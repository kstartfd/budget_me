"use client"

import { useCallback, useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import {
  IconBookmarkPlus,
  IconCheck,
  IconMessageCircle,
  IconMicrophone,
  IconRefresh,
  IconX,
} from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"
import { PathTopicChat } from "@/features/chat/components/path-topic-chat"
import { useLearning } from "@/features/learning/hooks"
import type { EntryInput, Topic } from "@/features/learning/schemas"
import { useTopicDetail, useTopicExerciseAttempt } from "../hooks"
import { topicSetComing, writerHasHeadStart } from "../progress"
import { LessonPanel } from "./lesson-panel"
import { PlayButton } from "./play-button"
import { Prose, RuleCard } from "./rule-card"
import { listenOnce, speechSupported } from "../speech"
import type { PathAttempt, PathExercise } from "../schemas"

/**
 * The writer ends a task with the word to work from in brackets, "I ... a book.
 * (buy)". It is a hint, not part of the task: it goes on a grey line under it.
 */
function splitHint(prompt: string): { task: string; hint: string } {
  const found = prompt.match(/^([\s\S]*\S)\s*\(([^()]{1,40})\)\s*$/)
  if (!found) return { task: prompt, hint: "" }

  return { task: found[1], hint: `(${found[2]})` }
}

const baseLanguage = (code: string) => code.split("-")[0].toLowerCase()

/** The notebook whose language pair is the program's: the chat with the
 * teacher is kept per notebook, so this is the one to open it on. */
function chatTopicFor(
  topics: Topic[] | undefined,
  language: string,
  translationLanguage: string,
): Topic | undefined {
  const sameLanguage = topics?.filter(
    (item) => baseLanguage(item.language) === baseLanguage(language),
  )

  return (
    sameLanguage?.find(
      (item) =>
        baseLanguage(item.translationLanguage) ===
        baseLanguage(translationLanguage),
    ) ?? sameLanguage?.[0]
  )
}

/** TopicRunner is the study screen of one topic: its exercises, checked one
 * answer at a time, with the agent's explanation when an answer is wrong. */
export function TopicRunner({ topicId }: { topicId: string }) {
  const t = useTranslations("path.topic")
  const { detail, generate, complete } = useTopicDetail(topicId)
  const { topics, saveEntry } = useLearning()
  const [away, setAway] = useState(false)
  const onPausedChange = useCallback((paused: boolean) => setAway(paused), [])

  if (detail.isPending)
    return (
      <p className="text-muted-foreground py-6 text-center text-sm">
        {t("loading")}
      </p>
    )
  if (detail.isError)
    return (
      <p role="alert" className="text-destructive py-6 text-center text-sm">
        {detail.error.message}
      </p>
    )

  const { path, topic, exercises, attempts } = detail.data
  const chatTopic = chatTopicFor(
    topics.data,
    path.language,
    path.translationLanguage,
  )
  const solved = new Set(
    attempts.filter((a) => a.correct).map((a) => a.exerciseId),
  )
  const currentExerciseIds = new Set(exercises.map((exercise) => exercise.id))
  const pastAttempts = attempts.filter(
    (attempt) => !currentExerciseIds.has(attempt.exerciseId),
  )
  const latestAttempts = new Map<string, PathAttempt>()
  for (const attempt of attempts) {
    latestAttempts.set(attempt.exerciseId, attempt)
  }

  return (
    <div className="space-y-3 p-4 pt-4">
      {detail.data.rule ? (
        // The rule comes first: what the topic is about is read before it is practised.
        <div className="space-y-3">
          <div className="bg-muted space-y-1 rounded-lg px-3.5 py-3">
            <p className="text-muted-foreground text-xs font-semibold tracking-wider uppercase">
              {t("goal")}
            </p>
            <p className="text-[13px] leading-relaxed">{topic.goal}</p>
          </div>
          <div className="border-border rounded-lg border p-4">
            <RuleCard rule={detail.data.rule} language={path.language} />
          </div>
        </div>
      ) : (
        <div className="bg-muted space-y-1 rounded-lg px-3.5 py-3">
          <p className="text-muted-foreground text-xs font-semibold tracking-wider uppercase">
            {t("rule")}
          </p>
          <p className="text-[13px] leading-relaxed">{topic.goal}</p>
          {topic.theory && <Prose text={topic.theory} className="text-sm" />}
        </div>
      )}
      {topic.status !== "completed" && (
        <LessonPanel
          topicId={topicId}
          onPausedChange={onPausedChange}
          // The set is written ahead of the learner; if it was not (the writer
          // failed, or it is an old program), starting the lesson writes it.
          // Right after the build the writer is still on its first sets, and
          // the topic polls for its own.
          onStart={() => {
            if (
              exercises.length === 0 &&
              !generate.isPending &&
              !writerHasHeadStart(path, detail.dataUpdatedAt)
            ) {
              generate.mutate(false)
            }
          }}
        />
      )}
      {topic.vocabulary.length > 0 && (
        <Vocabulary
          topicId={topicId}
          heading={t(topic.skill === "listening" || topic.skill === "speaking" ? "phrases" : "vocabulary")}
          words={topic.vocabulary}
          topicTitle={topic.title}
          notebookId={chatTopic?.id}
          saveEntry={saveEntry}
        />
      )}
      {pastAttempts.length > 0 && (
        <p className="text-muted-foreground text-xs">
          {t("pastAttempts", {
            count: pastAttempts.length,
            correct: pastAttempts.filter((attempt) => attempt.correct).length,
          })}
        </p>
      )}

      {generate.isError && (
        <p role="alert" className="text-destructive text-sm">
          {generate.error.message}
        </p>
      )}

      {away ? (
        <p className="text-muted-foreground bg-muted rounded-lg px-3.5 py-3 text-[13px]">
          {t("away")}
        </p>
      ) : exercises.length === 0 ? null : (
        <>
          {exercises.map((exercise, index) => (
            <ExerciseCard
              key={exercise.id}
              topicId={topicId}
              exercise={exercise}
              index={index + 1}
              total={exercises.length}
              topicTitle={topic.title}
              language={path.language}
              chatTopicId={chatTopic?.id}
              solved={solved.has(exercise.id)}
              latestAttempt={latestAttempts.get(exercise.id)}
            />
          ))}

          <div className="space-y-2 pt-1">
            {topic.status !== "completed" && complete.isError && (
              <p role="alert" className="text-destructive text-sm">
                {complete.error.message}
              </p>
            )}
            <div className="flex items-center gap-2">
              {topic.status !== "completed" && (
                <Button
                  variant="outline"
                  size="lg"
                  onClick={() => complete.mutate()}
                  disabled={complete.isPending || generate.isPending}
                  className="min-w-0 flex-1 text-[13px]"
                >
                  {t(complete.isPending ? "completing" : "complete")}
                </Button>
              )}
              <RefreshExercises
                generating={generate.isPending}
                onRefresh={() => generate.mutate(true)}
              />
            </div>
          </div>
        </>
      )}
      {!away && exercises.length === 0 && (
        <p
          role="status"
          className="text-muted-foreground bg-muted rounded-lg px-3.5 py-3 text-[13px]"
        >
          {t(
            generate.isPending
              ? "generating"
              : topicSetComing(path, 0, detail.dataUpdatedAt)
                ? "exercisesComing"
                : "exercisesOnStart",
          )}
        </p>
      )}
      <PathTopicChat
        topicId={topicId}
        topicTitle={topic.title}
        generalHref={
          chatTopic ? `/learning/chat?topic=${chatTopic.id}` : "/learning/chat"
        }
      />
    </div>
  )
}

/**
 * Asks for a new set of exercises: a small round button that stays out of the
 * way, since the set is written when the program is built and this is only for
 * a learner who wants another. The topic's count starts over with the new set.
 */
function RefreshExercises({
  generating,
  onRefresh,
}: {
  generating: boolean
  onRefresh: () => void
}) {
  const t = useTranslations("path.topic")

  return (
    <button
      type="button"
      onClick={onRefresh}
      disabled={generating}
      aria-label={t("refreshExercises")}
      title={t("refreshExercises")}
      className="border-border/60 bg-foreground/5 text-muted-foreground hover:bg-foreground/10 hover:text-foreground focus-visible:ring-ring/50 ml-auto inline-flex size-9 shrink-0 cursor-pointer items-center justify-center rounded-full border opacity-70 transition hover:opacity-100 focus-visible:ring-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-40"
    >
      <IconRefresh
        aria-hidden
        className={cn("size-4", generating && "animate-spin motion-reduce:animate-none")}
      />
    </button>
  )
}

function ExerciseCard({
  exercise,
  index,
  total,
  topicTitle,
  language,
  chatTopicId,
  solved,
  topicId,
  latestAttempt,
}: {
  topicId: string
  exercise: PathExercise
  index: number
  total: number
  topicTitle: string
  language: string
  chatTopicId?: string
  solved: boolean
  latestAttempt?: PathAttempt
}) {
  const t = useTranslations("path.topic")
  const attempt = useTopicExerciseAttempt(topicId)
  const { saveEntry } = useLearning()
  const [saved, setSaved] = useState(false)
  const [revealed, setRevealed] = useState(false)
  const [value, setValue] = useState("")
  const [last, setLast] = useState<PathAttempt | null>(null)
  const result = last ?? latestAttempt
  // A phrase to say is heard by the browser; where it cannot, or the
  // microphone is refused, the learner types it instead.
  const [hearing, setHearing] = useState(false)
  const [heard, setHeard] = useState("")
  const [voiceNote, setVoiceNote] = useState<"noSpeech" | null>(null)
  const [voiceFailed, setVoiceFailed] = useState<string | null>(null)
  const stopListening = useRef<(() => void) | null>(null)
  const voice = exercise.kind === "speak" && speechSupported() && voiceFailed === null

  useEffect(() => () => stopListening.current?.(), [])
  // The whole exercise goes to the teacher: the sentence or passage (the
  // prompt alone is just "fill in the gap"), the options, and what was answered.
  const exerciseText = [
    exercise.text,
    exercise.prompt,
    exercise.options.length > 0
      ? t("chatOptions", { options: exercise.options.join(" / ") })
      : "",
    result ? t("chatMyAnswer", { answer: result.answer }) : "",
  ]
    .filter(Boolean)
    .join("\n")
  const chat = new URLSearchParams({
    question: t("chatQuestion", { topic: topicTitle, prompt: exerciseText }),
  })
  if (chatTopicId) chat.set("topic", chatTopicId)

  const submit = (answer: string) => {
    if (!answer.trim()) return
    attempt.mutate(
      { exerciseId: exercise.id, answer },
      { onSuccess: (result) => setLast(result) },
    )
  }

  const startVoice = () => {
    setHeard("")
    setVoiceNote(null)
    setHearing(true)
    stopListening.current = listenOnce(language, {
      onText: (text) => {
        setHeard(text)
        submit(text)
      },
      onError: (reason) => {
        if (reason === "no-speech" || reason === "aborted") setVoiceNote("noSpeech")
        else setVoiceFailed(reason)
      },
      onEnd: () => setHearing(false),
    })
  }

  const { task, hint } = splitHint(exercise.prompt)
  const done = result ? result.correct : solved
  const failed = !!result && !result.correct
  // A word or a phrase worth keeping can go to the notebook once it is answered.
  const keepable = !!chatTopicId && !!result && isWordOrPhrase(exercise)

  return (
    <section className="border-border space-y-3 border-t-2 pt-5 first-of-type:border-t-0 first-of-type:pt-0">
      {exercise.text && !(exercise.kind === "listen" && !result) && (
        <p className="text-muted-foreground text-sm leading-relaxed italic">
          {exercise.text}
        </p>
      )}
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
        <span className="bg-muted text-muted-foreground rounded-full px-2.5 py-0.5 text-xs font-semibold tabular-nums">
          {index} / {total}
        </span>
        <p className="text-[15px] leading-snug font-semibold">{task}</p>
        {(done || failed) && (
          <span
            className={cn(
              "rounded-full border px-2 py-0.5 text-[11px] font-medium",
              done
                ? "border-success/30 bg-success/10 text-success"
                : "border-destructive/30 bg-destructive/10 text-destructive",
            )}
          >
            {t(done ? "solved" : "failed")}
          </span>
        )}
      </div>

      {hint && <p className="text-muted-foreground -mt-1.5 text-[13px]">{hint}</p>}

      {exercise.kind === "choice" || exercise.kind === "listen" ? (
        <div className="space-y-2">
          {exercise.kind === "listen" && (
            <PlayButton path={`/path/exercises/${exercise.id}/audio`} label={t("hear")} />
          )}
          <div
            className={cn(
              "grid gap-2",
              // Short options sit two by two, long ones one under another.
              exercise.options.every((option) => option.length <= 22)
                ? "grid-cols-2"
                : "sm:grid-cols-2",
            )}
          >
          {exercise.options.map((option) => {
            const picked = result?.answer === option
            const right = picked && result.correct
            const wrong = picked && !result.correct
            const reveal =
              failed && !!exercise.answer && option === exercise.answer

            return (
              <button
                key={option}
                type="button"
                onClick={() => submit(option)}
                disabled={attempt.isPending}
                className={cn(
                  "border-input bg-background hover:bg-muted/50 flex min-h-10 w-full cursor-pointer items-center gap-3 rounded-md border px-3.5 py-2 text-left text-sm transition-colors disabled:cursor-not-allowed",
                  right && "border-success bg-success/10 text-success font-semibold",
                  wrong &&
                    "border-destructive bg-destructive/10 text-destructive font-semibold",
                  reveal && "border-success text-success border-dashed font-semibold",
                )}
              >
                <span
                  className={cn(
                    "border-input size-4 shrink-0 rounded-full border",
                    right && "border-success bg-success",
                    wrong && "border-destructive bg-destructive",
                    reveal && "border-success",
                  )}
                />
                <span className="min-w-0 flex-1">{option}</span>
                {(right || reveal) && (
                  <IconCheck size={14} className="shrink-0" />
                )}
                {wrong && <IconX size={14} className="shrink-0" />}
              </button>
            )
          })}
          </div>
        </div>
      ) : voice ? (
        <div className="space-y-2.5">
          <p className="bg-muted rounded-lg px-3.5 py-3 text-[15px] leading-snug font-medium">
            {exercise.answer}
          </p>
          <div className="flex flex-wrap gap-2">
            <PlayButton path={`/path/exercises/${exercise.id}/audio`} label={t("hear")} />
            <Button
              type="button"
              size="lg"
              onClick={startVoice}
              disabled={hearing || attempt.isPending}
              className="text-[13px]"
            >
              <IconMicrophone size={16} className={hearing ? "animate-pulse" : undefined} />
              {t(hearing ? "listening" : "speakNow")}
            </Button>
          </div>
          {heard && (
            <p className="text-muted-foreground text-xs">
              {t("heard")}: {heard}
            </p>
          )}
          {voiceNote && (
            <p className="text-muted-foreground text-xs">{t("noSpeech")}</p>
          )}
        </div>
      ) : (
        <form
          className={cn(
            "flex gap-2",
            exercise.kind === "write" ? "flex-wrap" : "flex-col sm:flex-row",
          )}
          onSubmit={(e) => {
            e.preventDefault()
            submit(value)
          }}
        >
          {exercise.kind === "write" ? (
            <Textarea
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={t("writePlaceholder")}
              aria-label={exercise.prompt}
              rows={3}
              className="bg-background basis-full text-sm md:text-sm"
            />
          ) : (
            <Input
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={t(
                exercise.kind === "fill"
                  ? "fillPlaceholder"
                  : exercise.kind === "speak"
                    ? "speakFallback"
                    : "translatePlaceholder",
              )}
              aria-label={exercise.prompt}
              className="bg-background h-10 text-sm md:text-sm"
            />
          )}
          <div className="flex w-full gap-2 sm:w-auto">
            <Button
              type="submit"
              // Outlined and dull while there is nothing to check; it fills in when there is.
              variant={value.trim() ? "default" : "outline"}
              disabled={attempt.isPending || !value.trim()}
              className="h-10 flex-1 px-5 text-sm sm:flex-none"
            >
              {t(attempt.isPending ? "checking" : "check")}
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={() => setRevealed((open) => !open)}
              aria-expanded={revealed}
              className="text-muted-foreground h-10 px-3 text-sm underline underline-offset-4 hover:bg-transparent"
            >
              {t(revealed ? "hideAnswer" : "showAnswer")}
            </Button>
          </div>
        </form>
      )}
      {exercise.kind === "speak" && voiceFailed !== null && (
        <p className="text-muted-foreground text-xs">{t("voiceFailed")}</p>
      )}
      {exercise.kind === "write" && result && (
        <div className="bg-muted space-y-1 rounded-lg px-3.5 py-3 text-[13px] leading-relaxed">
          <p className="text-muted-foreground text-xs font-semibold tracking-wider uppercase">
            {t("sampleAnswer")}
          </p>
          <p>{exercise.answer}</p>
        </div>
      )}
      {revealed && exercise.kind !== "choice" && exercise.kind !== "listen" && (
        <div className="bg-muted space-y-1 rounded-lg px-3.5 py-3 text-[13px] leading-relaxed">
          <p>
            <span className="text-muted-foreground">{t("correctAnswer")}: </span>
            <strong>{exercise.answer}</strong>
          </p>
          {exercise.rule && <p className="text-muted-foreground">{exercise.rule}</p>}
        </div>
      )}

      {attempt.isError && (
        <p role="alert" className="text-destructive text-sm">
          {attempt.error.message}
        </p>
      )}
      {result && (
        <div
          className={cn(
            "space-y-1 rounded-lg border px-3.5 py-3 text-sm",
            result.correct
              ? "border-success/30 bg-success/10"
              : "border-destructive/30 bg-destructive/10",
          )}
        >
          <p
            className={cn(
              "font-semibold",
              result.correct ? "text-success" : "text-destructive",
            )}
          >
            {result.correct ? t("correct") : t("wrong")}
          </p>
          <p className="text-foreground text-[13px] leading-relaxed">
            {t("myAnswer")}: {result.answer}.
            {!result.correct && result.feedback && (
              <>
                {" "}
                <QuotedEmphasis text={result.feedback} />
              </>
            )}
          </p>
          <div className="flex flex-wrap gap-2">
            {!result.correct && (
              <Button
                asChild
                variant="outline"
                size="lg"
                className="bg-background mt-2 text-[13px]"
              >
                <Link href={`/learning/chat?${chat}`}>
                  <IconMessageCircle size={14} />
                  {t("askTeacher")}
                </Link>
              </Button>
            )}
            {keepable && (
              <Button
                type="button"
                variant="outline"
                size="lg"
                disabled={saved || saveEntry.isPending}
                onClick={async () => {
                  await saveEntry.mutateAsync(
                    exerciseEntry(exercise, topicTitle, chatTopicId),
                  )
                  setSaved(true)
                }}
                className="bg-background mt-2 text-[13px]"
              >
                {saved ? <IconCheck size={14} /> : <IconBookmarkPlus size={14} />}
                {saved
                  ? t("savedToNotebook")
                  : t("saveToNotebook", { text: exercise.answer })}
              </Button>
            )}
          </div>
          {saveEntry.isError && (
            <p role="alert" className="text-destructive text-xs">
              {saveEntry.error.message}
            </p>
          )}
        </div>
      )}
    </section>
  )
}

function QuotedEmphasis({ text }: { text: string }) {
  const parts = text.split(/(["“«])([^"”»\n]+)(["”»])/g)

  return (
    <>
      {parts.map((part, index) => {
        if (index % 4 === 2) return <strong key={index}>{part}</strong>
        return part
      })}
    </>
  )
}

type SaveEntry = ReturnType<typeof useLearning>["saveEntry"]

/** Vocabulary lists the key words of a topic; each goes to the notebook of
 * the language pair in one tap, and the whole list in one more. */
function Vocabulary({
  topicId,
  heading,
  words,
  topicTitle,
  notebookId,
  saveEntry,
}: {
  topicId: string
  heading: string
  words: { word: string; translation: string }[]
  topicTitle: string
  notebookId?: string
  saveEntry: SaveEntry
}) {
  const t = useTranslations("path.topic")
  const [saved, setSaved] = useState<Set<string>>(new Set())

  const save = async (word: { word: string; translation: string }) => {
    if (!notebookId || saved.has(word.word)) return
    await saveEntry.mutateAsync(entryOf(word, topicTitle, notebookId))
    setSaved((old) => new Set(old).add(word.word))
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-3">
        <p className="text-muted-foreground text-xs font-semibold tracking-wider uppercase">
          {heading}
        </p>
        {notebookId && (
          <Button
            variant="ghost"
            size="sm"
            disabled={
              saveEntry.isPending || words.every((w) => saved.has(w.word))
            }
            onClick={async () => {
              for (const word of words) await save(word)
            }}
            className="h-auto px-0 py-0 text-xs"
          >
            {t("saveAll")}
          </Button>
        )}
      </div>
      <ul className="space-y-1">
        {words.map((word, index) => (
          <li
            key={word.word}
            className="border-border flex items-center gap-2 rounded-md border px-3 py-1.5 text-[13px]"
          >
            <PlayButton
              compact
              path={`/path/topics/${topicId}/words/${index}/audio`}
              label={t("hearWord", { word: word.word })}
            />
            <span className="font-medium">{word.word}</span>
            <span className="text-muted-foreground min-w-0 flex-1 truncate">
              {word.translation}
            </span>
            {notebookId && (
              <button
                type="button"
                aria-label={t("save")}
                disabled={saveEntry.isPending || saved.has(word.word)}
                onClick={() => void save(word)}
                className="text-muted-foreground hover:text-foreground cursor-pointer disabled:opacity-40"
              >
                {saved.has(word.word) ? (
                  <IconCheck size={15} />
                ) : (
                  <IconBookmarkPlus size={15} />
                )}
              </button>
            )}
          </li>
        ))}
      </ul>
      {saveEntry.isError && (
        <p role="alert" className="text-destructive text-xs">
          {saveEntry.error.message}
        </p>
      )}
    </div>
  )
}

const maxPhraseWords = 4

/** A choice or fill exercise whose answer is a word or a short phrase. A
 * translation is a whole sentence, which is not a notebook entry. */
function isWordOrPhrase(exercise: PathExercise): boolean {
  const answer = exercise.answer.trim()
  if (!["choice", "fill", "speak"].includes(exercise.kind) || !answer) return false
  if (/[.!?;:]$/.test(answer)) return false

  return answer.split(/\s+/).length <= maxPhraseWords
}

/** The notebook entry of an exercise's answer: the exercise sentence with the
 * gap filled becomes the example, the rule is kept with it. */
function exerciseEntry(
  exercise: PathExercise,
  topicTitle: string,
  notebookId: string,
): EntryInput {
  const answer = exercise.answer.trim()
  const gap = /_{2,}|\.{3}|…/
  const example = gap.test(exercise.prompt)
    ? exercise.prompt.replace(gap, answer)
    : exercise.prompt

  return {
    topicId: notebookId,
    kind: answer.includes(" ") ? "phrase" : "word",
    text: answer,
    transcription: "",
    translation: "",
    explanation: "",
    nativeUsage: "",
    example: exercise.kind === "choice" || exercise.kind === "fill" ? example : "",
    rule: exercise.rule,
    source: topicTitle,
    status: "inbox",
  }
}

/** The notebook entry of a word: new, in the inbox, remembering where it came from. */
function entryOf(
  word: { word: string; translation: string },
  topicTitle: string,
  topicId: string,
): EntryInput {
  return {
    topicId,
    kind: word.word.trim().includes(" ") ? "phrase" : "word",
    text: word.word,
    transcription: "",
    translation: word.translation,
    explanation: "",
    nativeUsage: "",
    example: "",
    rule: "",
    source: topicTitle,
    status: "inbox",
  }
}
