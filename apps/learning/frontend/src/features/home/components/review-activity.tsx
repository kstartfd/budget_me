"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useAuth } from "@/features/auth";
import { submitAttempt } from "@/features/path/api";
import { PlayButton } from "@/features/path/components/play-button";
import type { LoopActivity } from "@/features/path/schemas";

/** Answer one due learning-loop review. The activity ID makes the answer
 * independent evidence and closes that review in the backend transaction. */
export function ReviewActivity({
  activity,
  onClose,
}: {
  activity: LoopActivity;
  onClose: () => void;
}) {
  const t = useTranslations("home.dash");
  const { user } = useAuth();
  const client = useQueryClient();
  const exercise = activity.exercise;
  const [answer, setAnswer] = useState("");
  const [parts, setParts] = useState<string[]>([]);
  const [result, setResult] = useState<{
    correct: boolean;
    feedback: string;
  } | null>(null);
  const attempt = useMutation({
    mutationFn: (value: string) =>
      submitAttempt(exercise!.id, value, activity.id),
    onSuccess: async (saved) => {
      setResult(saved);
      await client.invalidateQueries({ queryKey: ["path", user?.id] });
    },
  });

  if (!exercise) return null;

  const kind = exercise.kind;
  const gaps =
    kind === "cloze"
      ? Math.floor(exercise.text.split(/\[(\d{1,2})\]/).length / 2)
      : kind === "read"
        ? exercise.options.length
        : 0;
  const values = Array.from({ length: gaps }, (_, i) => parts[i] ?? "");
  const value =
    gaps > 0 ? values.map((part) => part.trim()).join(" | ") : answer.trim();
  const ready =
    gaps > 0
      ? values.every((part) => part.trim().length > 0)
      : value.length > 0;
  const setPart = (index: number, next: string) =>
    setParts((previous) =>
      Array.from({ length: gaps }, (_, i) =>
        i === index ? next : (previous[i] ?? ""),
      ),
    );

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent className="max-h-[calc(100dvh-3rem)] overflow-y-auto p-6 sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle className="text-lg">
            {activity.title || t("reviewTitle")}
          </DialogTitle>
          <DialogDescription>{t("reviewPrompt")}</DialogDescription>
        </DialogHeader>
        {result ? (
          <div className="space-y-4">
            <p
              role="status"
              className={
                result.correct
                  ? "font-medium text-success"
                  : "font-medium text-destructive"
              }
            >
              {t(result.correct ? "reviewCorrect" : "reviewIncorrect")}
            </p>
            {result.feedback && (
              <p className="bg-muted rounded-lg p-3 text-sm">
                {result.feedback}
              </p>
            )}
            <Button onClick={onClose}>{t("reviewBack")}</Button>
          </div>
        ) : (
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault();
              if (ready && !attempt.isPending) attempt.mutate(value);
            }}
          >
            {exercise.text && kind !== "listen" && kind !== "order" && (
              <p className="text-muted-foreground whitespace-pre-wrap text-sm">
                {exercise.text}
              </p>
            )}
            {exercise.prompt && (
              <p className="text-sm font-medium">{exercise.prompt}</p>
            )}
            {kind === "listen" && (
              <PlayButton
                path={`/path/exercises/${exercise.id}/audio`}
                label={t("reviewListen")}
              />
            )}
            {(kind === "choice" || kind === "listen") &&
            exercise.options.length > 0 ? (
              <fieldset className="grid gap-2">
                <legend className="sr-only">{t("reviewAnswer")}</legend>
                {exercise.options.map((option) => (
                  <label
                    key={option}
                    className="border-border flex cursor-pointer items-center gap-2 rounded-lg border p-3 text-sm"
                  >
                    <input
                      type="radio"
                      name="review-answer"
                      value={option}
                      checked={answer === option}
                      onChange={() => setAnswer(option)}
                    />
                    {option}
                  </label>
                ))}
              </fieldset>
            ) : kind === "cloze" && gaps > 0 ? (
              <div className="space-y-2">
                {Array.from({ length: gaps }, (_, i) => (
                  <label key={i} className="block space-y-1 text-sm">
                    <span>{t("reviewGap", { n: i + 1 })}</span>
                    <Input
                      value={values[i]}
                      onChange={(event) => setPart(i, event.target.value)}
                      autoComplete="off"
                    />
                  </label>
                ))}
              </div>
            ) : kind === "read" && gaps > 0 ? (
              <div className="space-y-3">
                {exercise.options.map((question, i) => (
                  <label key={i} className="block space-y-1 text-sm">
                    <span>{question}</span>
                    <Input
                      value={values[i]}
                      onChange={(event) => setPart(i, event.target.value)}
                      autoComplete="off"
                    />
                  </label>
                ))}
              </div>
            ) : (
              <div className="space-y-2">
                {kind === "order" && (
                  <p className="text-muted-foreground text-sm">
                    {exercise.options.join(" · ")}
                  </p>
                )}
                <label className="block space-y-1 text-sm">
                  <span>{t("reviewAnswer")}</span>
                  {kind === "write" || kind === "translate" ? (
                    <Textarea
                      value={answer}
                      onChange={(event) => setAnswer(event.target.value)}
                      maxLength={2000}
                    />
                  ) : (
                    <Input
                      value={answer}
                      onChange={(event) => setAnswer(event.target.value)}
                      maxLength={2000}
                      autoComplete="off"
                    />
                  )}
                </label>
              </div>
            )}
            {attempt.isError && (
              <p role="alert" className="text-destructive text-sm">
                {attempt.error.message}
              </p>
            )}
            <div className="flex justify-end">
              <Button type="submit" disabled={!ready || attempt.isPending}>
                {t(attempt.isPending ? "reviewChecking" : "reviewCheck")}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
