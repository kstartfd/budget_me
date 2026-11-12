"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { IconPlus, IconTrash } from "@tabler/icons-react";

import { SectionKicker, sectionCard } from "@/components/section";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

import {
  normalizeWord,
  triggerColors,
  triggerStyles,
  useResetTriggerSettings,
  useSaveTriggerSettings,
  useTriggerSettings,
  type Trigger,
  type TriggerColor,
  type TriggerStyle,
} from "../triggers";
import { Labelled, MessageBody } from "./message-body";

const selectClass =
  "border-input bg-background h-9 w-full rounded-md border px-2 text-sm";

const sample = [
  "Rule: use Present Simple for habits.",
  "Example: She drinks coffee every morning.",
  "Remember: the verb takes -s after he, she, it.",
  "A: What do you do?\nB: I work in an office.",
].join("\n\n");

function StyleSelect({
  value,
  onChange,
}: {
  value: TriggerStyle;
  onChange: (value: TriggerStyle) => void;
}) {
  const t = useTranslations("chat.triggers");

  return (
    <select
      aria-label={t("style")}
      className={selectClass}
      value={value}
      onChange={(e) => onChange(e.target.value as TriggerStyle)}
    >
      {triggerStyles.map((s) => (
        <option key={s} value={s}>
          {t(`styles.${s}`)}
        </option>
      ))}
    </select>
  );
}

function ColorSelect({
  value,
  onChange,
}: {
  value: TriggerColor;
  onChange: (value: TriggerColor) => void;
}) {
  const t = useTranslations("chat.triggers");

  return (
    <select
      aria-label={t("color")}
      className={selectClass}
      value={value}
      onChange={(e) => onChange(e.target.value as TriggerColor)}
    >
      {triggerColors.map((c) => (
        <option key={c} value={c}>
          {t(`colors.${c}`)}
        </option>
      ))}
    </select>
  );
}

/** The table of words the chat marks in a reply, and how it marks each. */
export function TriggersSection() {
  const t = useTranslations("chat.triggers");
  const settings = useTriggerSettings();
  const save = useSaveTriggerSettings();
  const reset = useResetTriggerSettings();
  const [word, setWord] = useState("");
  const [style, setStyle] = useState<TriggerStyle>("frame");
  const [color, setColor] = useState<TriggerColor>("blue");

  const update = (triggers: Trigger[]) =>
    save.mutate({ ...settings, triggers });

  const add = () => {
    const w = normalizeWord(word);
    if (!w || settings.triggers.some((x) => normalizeWord(x.word) === w))
      return;

    update([
      ...settings.triggers,
      { id: crypto.randomUUID(), word: w, style, color },
    ]);
    setWord("");
  };

  const patch = (id: string, change: Partial<Trigger>) =>
    update(
      settings.triggers.map((x) => (x.id === id ? { ...x, ...change } : x)),
    );

  return (
    <section className="flex flex-col gap-3">
      <SectionKicker>{t("title")}</SectionKicker>

      <div className={cn(sectionCard, "space-y-4 p-4")}>
        <p className="text-sm leading-[1.45]">{t("hint")}</p>

        <div className="overflow-x-auto">
          <table className="w-full min-w-[34rem] border-separate border-spacing-y-1.5 text-sm">
            <thead>
              <tr className="text-muted-foreground text-left text-xs font-semibold">
                <th className="px-1 font-semibold">{t("word")}</th>
                <th className="px-1 font-semibold">{t("style")}</th>
                <th className="px-1 font-semibold">{t("color")}</th>
                <th className="px-1 font-semibold">{t("look")}</th>
                <th className="w-10" />
              </tr>
            </thead>
            <tbody>
              {settings.triggers.map((x) => (
                <tr key={x.id} className="align-middle">
                  <td className="px-1 font-semibold break-words">{x.word}</td>
                  <td className="px-1">
                    <StyleSelect
                      value={x.style}
                      onChange={(s) => patch(x.id, { style: s })}
                    />
                  </td>
                  <td className="px-1">
                    <ColorSelect
                      value={x.color}
                      onChange={(c) => patch(x.id, { color: c })}
                    />
                  </td>
                  <td className="min-w-40 px-1">
                    <Labelled label={x.word} text={t("lookText")} trigger={x} />
                  </td>
                  <td className="px-1 text-right">
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={t("remove")}
                      onClick={() =>
                        update(settings.triggers.filter((y) => y.id !== x.id))
                      }
                    >
                      <IconTrash size={16} />
                    </Button>
                  </td>
                </tr>
              ))}
              {settings.triggers.length === 0 && (
                <tr>
                  <td
                    colSpan={5}
                    className="text-muted-foreground px-1 py-3 text-center"
                  >
                    {t("empty")}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <form
          className="flex flex-wrap items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            add();
          }}
        >
          <Input
            value={word}
            maxLength={40}
            placeholder={t("placeholder")}
            aria-label={t("placeholder")}
            className="min-w-32 flex-1"
            onChange={(e) => setWord(e.target.value)}
          />
          <div className="w-36">
            <StyleSelect value={style} onChange={setStyle} />
          </div>
          <div className="w-36">
            <ColorSelect value={color} onChange={setColor} />
          </div>
          <Button type="submit" disabled={normalizeWord(word) === ""}>
            <IconPlus size={16} />
            {t("add")}
          </Button>
        </form>

        <label className="flex items-center justify-between gap-3 text-sm">
          {t("anyLabel")}
          <Switch
            checked={settings.anyLabel}
            onCheckedChange={(anyLabel) =>
              save.mutate({ ...settings, anyLabel })
            }
          />
        </label>
        <label className="flex items-center justify-between gap-3 text-sm">
          {t("dialogue")}
          <Switch
            checked={settings.dialogue}
            onCheckedChange={(dialogue) =>
              save.mutate({ ...settings, dialogue })
            }
          />
        </label>

        <div className="bg-muted border-border/60 rounded-2xl border px-4 py-3 text-[15px] leading-relaxed">
          <MessageBody text={sample} />
        </div>

        {save.isError && (
          <p role="alert" className="text-destructive text-sm">
            {t("saveFailed")}
          </p>
        )}

        <Button
          variant="outline"
          disabled={reset.isPending}
          onClick={() => reset.mutate()}
        >
          {t("reset")}
        </Button>
      </div>
    </section>
  );
}
