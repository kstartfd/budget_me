"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";

import { useAuth } from "@/features/auth";
import { call } from "@/features/learning/api";

/**
 * The learner's dictionary of trigger words: what the chat screen looks for in
 * a teacher's reply and how it marks it. It is kept on the server, per user,
 * so it follows them to every device; until they change anything the defaults
 * below apply.
 *
 * A word is matched two ways:
 *  - at the start of a paragraph followed by a colon ("Rule: ...") it marks
 *    the whole paragraph with a block style (frame, bar);
 *  - anywhere else, as a whole word, it marks just that word with an inline
 *    style (marker, chip, bold, underline).
 * Each trigger has one style; a block style used inline falls back to a chip.
 */

export const triggerStyles = [
  "frame",
  "bar",
  "marker",
  "chip",
  "bold",
  "underline",
] as const;
export type TriggerStyle = (typeof triggerStyles)[number];

export const blockStyles: readonly TriggerStyle[] = ["frame", "bar"];

export const triggerColors = [
  "neutral",
  "blue",
  "green",
  "amber",
  "red",
] as const;
export type TriggerColor = (typeof triggerColors)[number];

const triggerSchema = z.object({
  id: z.string(),
  word: z.string(),
  style: z.enum(triggerStyles),
  color: z.enum(triggerColors),
});
export type Trigger = z.infer<typeof triggerSchema>;

const settingsSchema = z.object({
  /** Any short line ending with a colon ("Present Simple:") is a label too. */
  anyLabel: z.boolean(),
  /** "A: ..." / "B: ..." lines are shown as a dialogue. */
  dialogue: z.boolean(),
  triggers: z.array(triggerSchema),
});
export type TriggerSettings = z.infer<typeof settingsSchema>;

export const defaultSettings: TriggerSettings = {
  anyLabel: true,
  dialogue: true,
  triggers: [
    ["правило", "frame", "blue"],
    ["rule", "frame", "blue"],
    ["пример", "bar", "green"],
    ["example", "bar", "green"],
    ["упражнение", "frame", "amber"],
    ["exercise", "frame", "amber"],
    ["запомни", "marker", "amber"],
    ["remember", "marker", "amber"],
    ["важно", "marker", "red"],
    ["important", "marker", "red"],
  ].map(([word, style, color]) => ({
    id: `default-${word}`,
    word,
    style: style as TriggerStyle,
    color: color as TriggerColor,
  })),
};

/** `null` until the user has changed anything. */
const fetchSettings = () =>
  call("/chat/marks", "GET", (d) => settingsSchema.nullable().parse(d));

const saveSettings = (input: TriggerSettings) =>
  call("/chat/marks", "PUT", (d) => settingsSchema.parse(d), input);

const deleteSettings = () => call("/chat/marks", "DELETE", () => undefined);

function settingsKey(userId: string | undefined) {
  return ["learning", userId, "chat-marks"] as const;
}

/** The user's dictionary, or the defaults while it loads or is untouched. */
export function useTriggerSettings(): TriggerSettings {
  const { user } = useAuth();
  const { data } = useQuery({
    queryKey: settingsKey(user?.id),
    queryFn: fetchSettings,
    enabled: !!user,
  });

  return data ?? defaultSettings;
}

/** Saves the whole dictionary; the screen shows the change at once. */
export function useSaveTriggerSettings() {
  const { user } = useAuth();
  const client = useQueryClient();
  const key = settingsKey(user?.id);

  return useMutation({
    mutationFn: saveSettings,
    onMutate: async (next) => {
      await client.cancelQueries({ queryKey: key });
      const previous = client.getQueryData(key);
      client.setQueryData(key, next);

      return { previous };
    },
    onError: (_error, _next, context) =>
      client.setQueryData(key, context?.previous),
    onSuccess: (saved) => client.setQueryData(key, saved),
  });
}

/** Back to the defaults: the stored dictionary is forgotten. */
export function useResetTriggerSettings() {
  const { user } = useAuth();
  const client = useQueryClient();

  return useMutation({
    mutationFn: deleteSettings,
    onSuccess: () => client.setQueryData(settingsKey(user?.id), null),
  });
}

/** Words are compared case-insensitively and without a trailing colon. */
export function normalizeWord(word: string) {
  return word
    .trim()
    .replace(/[:：]+$/, "")
    .trim()
    .toLowerCase();
}

export function findTrigger(settings: TriggerSettings, word: string) {
  const w = normalizeWord(word);
  return settings.triggers.find((t) => normalizeWord(t.word) === w);
}

const escape = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

/** A whole-word, case-insensitive pattern over every trigger word, or null. */
export function inlineTriggerPattern(settings: TriggerSettings) {
  const words = settings.triggers
    .map((t) => normalizeWord(t.word))
    .filter(Boolean)
    .sort((a, b) => b.length - a.length);
  if (words.length === 0) return null;

  return new RegExp(
    `(?<![\\p{L}\\p{N}_])(${words.map(escape).join("|")})(?![\\p{L}\\p{N}_])`,
    "giu",
  );
}
