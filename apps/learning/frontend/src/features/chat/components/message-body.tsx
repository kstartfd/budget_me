import { Example } from "@/components/example";
import { InlineMarkdown } from "@/components/inline-markdown";
import { cn } from "@/lib/utils";

import {
  blockStyles,
  findTrigger,
  inlineTriggerPattern,
  useTriggerSettings,
  type Trigger,
  type TriggerColor,
  type TriggerSettings,
} from "../triggers";

/**
 * Turns a teacher's plain-text reply into readable blocks. The model is only
 * asked for a few plain conventions (see the backend prompt), so this never
 * needs a full Markdown renderer:
 *
 *  - a paragraph starting with ✏️ is a correction card; inside it ❌ / ✅ lines
 *    are the wrong and the right wording;
 *  - a paragraph starting with "Word:" is marked by the learner's dictionary of
 *    trigger words (frame, bar, ...); any other short "Label:" is a neutral
 *    frame when that is switched on;
 *  - "A: ..." / "B: ..." lines are a dialogue;
 *  - "1. ..." lines are exercises; ___ in them is a gap, (go) is its hint;
 *  - "- ..." lines are a list;
 *  - trigger words anywhere else, "quoted", «quoted» and `code` fragments are
 *    marked inline.
 */

type Block =
  | { type: "paragraph"; text: string }
  | { type: "labelled"; label: string; text: string; trigger?: Trigger }
  | { type: "list"; ordered: boolean; items: string[] }
  | { type: "correction"; lines: string[] }
  | { type: "dialogue"; turns: { speaker: string; text: string }[] };

const orderedItem = /^\s*(\d{1,2})[.)]\s+(.*)$/;
const bulletItem = /^\s*[-•*]\s+(.*)$/;
const speakerLine = /^\s*([\p{L}][\p{L}\p{N}_ ]{0,15}?)\s*:\s+(.+)$/u;

/** "Present Simple:" - a short head: 1-4 words, no sentence punctuation. */
const labelHead = /^\s*([^\n:：.,!?;()]{1,40}?)\s*[:：]\s*([\s\S]*)$/;

function asDialogue(lines: string[]): Block | null {
  const turns = lines.map((l) => speakerLine.exec(l));
  if (lines.length < 2 || turns.some((m) => !m)) return null;

  const rows = turns.map((m) => ({ speaker: m![1], text: m![2] }));
  return new Set(rows.map((r) => r.speaker.toLowerCase())).size >= 2
    ? { type: "dialogue", turns: rows }
    : null;
}

export function parseMessage(text: string, settings: TriggerSettings): Block[] {
  const blocks: Block[] = [];

  for (const raw of text.trim().split(/\n{2,}/)) {
    const lines = raw.split("\n").filter((l) => l.trim() !== "");
    if (lines.length === 0) continue;

    if (lines[0].trimStart().startsWith("✏")) {
      blocks.push({ type: "correction", lines });
      continue;
    }

    if (settings.dialogue) {
      const dialogue = asDialogue(lines);
      if (dialogue) {
        blocks.push(dialogue);
        continue;
      }
    }

    // A paragraph may be a mix of prose and list lines: split at the edges.
    let prose: string[] = [];
    let list: { ordered: boolean; items: string[] } | null = null;

    const flushProse = () => {
      if (prose.length === 0) return;
      const joined = prose.join("\n");
      prose = [];

      const head = labelHead.exec(joined);
      const words = head?.[1].trim().split(/\s+/).length ?? 0;
      const trigger = head ? findTrigger(settings, head[1]) : undefined;

      if (head && (trigger || (settings.anyLabel && words <= 4))) {
        blocks.push({
          type: "labelled",
          label: head[1].trim(),
          text: head[2],
          trigger,
        });
        return;
      }
      blocks.push({ type: "paragraph", text: joined });
    };
    const flushList = () => {
      if (list) blocks.push({ type: "list", ...list });
      list = null;
    };

    for (const line of lines) {
      const o = orderedItem.exec(line);
      const b = o ? null : bulletItem.exec(line);
      const item = o?.[2] ?? b?.[1];

      if (item === undefined) {
        flushList();
        prose.push(line);
        continue;
      }

      flushProse();
      const ordered = !!o;
      if (list && (list as { ordered: boolean }).ordered !== ordered)
        flushList();
      if (!list) list = { ordered, items: [] };
      (list as { items: string[] }).items.push(item);
    }
    flushList();
    flushProse();
  }

  return blocks;
}

// One class set per colour. Full class names, so Tailwind can see them.
const palette: Record<
  TriggerColor,
  { frame: string; bar: string; marker: string; label: string }
> = {
  neutral: {
    frame: "border-border bg-background",
    bar: "border-foreground/50",
    marker: "bg-foreground/10",
    label: "text-muted-foreground",
  },
  blue: {
    frame: "border-blue-500/40 bg-blue-500/5",
    bar: "border-blue-500",
    marker: "bg-blue-500/20",
    label: "text-blue-700 dark:text-blue-300",
  },
  green: {
    frame: "border-green-500/40 bg-green-500/5",
    bar: "border-green-500",
    marker: "bg-green-500/20",
    label: "text-green-700 dark:text-green-300",
  },
  amber: {
    frame: "border-amber-500/40 bg-amber-500/5",
    bar: "border-amber-500",
    marker: "bg-amber-500/25",
    label: "text-amber-700 dark:text-amber-300",
  },
  red: {
    frame: "border-red-500/40 bg-red-500/5",
    bar: "border-red-500",
    marker: "bg-red-500/20",
    label: "text-red-700 dark:text-red-300",
  },
};

/** A trigger word marked inline according to its style. */
export function Mark({
  trigger,
  children,
}: {
  trigger: Trigger;
  children: string;
}) {
  const colors = palette[trigger.color];

  switch (trigger.style) {
    case "marker":
      return (
        <mark className={cn("rounded px-1 text-inherit", colors.marker)}>
          {children}
        </mark>
      );
    case "bold":
      return <strong className={colors.label}>{children}</strong>;
    case "underline":
      return (
        <span
          className={cn(
            "underline decoration-2 underline-offset-4",
            colors.label,
          )}
        >
          {children}
        </span>
      );
    default:
      // chip, and the block styles when they are met mid-sentence
      return (
        <span
          className={cn(
            "rounded-md border px-1.5 py-0.5 text-[0.93em] font-semibold",
            colors.frame,
            colors.label,
          )}
        >
          {children}
        </span>
      );
  }
}

// «...», “...”, "..." and `...`. A lone apostrophe (don't) is never a quote.
const quotePattern = /(«[^»\n]+»|“[^”\n]+”|"[^"\n]+"|`[^`\n]+`)/g;

const chipClass =
  "bg-background border-border rounded-md border px-1.5 py-0.5 text-[0.93em] font-medium";

/** Plain text with the dictionary's trigger words marked, then emphasis. */
function Marked({ text }: { text: string }) {
  const settings = useTriggerSettings();
  const pattern = inlineTriggerPattern(settings);
  if (!pattern) return <InlineMarkdown text={text} />;

  return (
    <>
      {text.split(pattern).map((part, i) => {
        if (i % 2 === 0) return <InlineMarkdown key={i} text={part} />;
        const trigger = findTrigger(settings, part);
        return trigger ? (
          <Mark key={i} trigger={trigger}>
            {part}
          </Mark>
        ) : (
          part
        );
      })}
    </>
  );
}

export function Inline({ text }: { text: string }) {
  return (
    <>
      {text.split(quotePattern).map((part, i) => {
        if (i % 2 === 0) return <Marked key={i} text={part} />;
        if (part.startsWith("`")) {
          return <Example key={i}>{part.slice(1, -1)}</Example>;
        }
        return (
          <q
            key={i}
            className={cn(chipClass, "before:content-none after:content-none")}
          >
            <InlineMarkdown text={part.slice(1, -1)} />
          </q>
        );
      })}
    </>
  );
}

const gapPattern = /(_{2,}|\([^()\n]{1,24}\))/g;

/** An exercise line: gaps become blanks, a (hint) in brackets becomes a chip. */
function Exercise({ text }: { text: string }) {
  if (!/_{2,}/.test(text)) return <Inline text={text} />;

  return (
    <>
      {text.split(gapPattern).map((part, i) => {
        if (i % 2 === 0) return <Inline key={i} text={part} />;
        if (part.startsWith("_")) {
          return (
            <span
              key={i}
              aria-label="gap"
              className="border-foreground/50 mx-0.5 inline-block w-14 border-b-2 align-baseline"
            />
          );
        }
        return (
          <span
            key={i}
            className="bg-secondary border-border mx-0.5 rounded-md border px-2 py-0.5 text-[0.93em] font-semibold"
          >
            {part.slice(1, -1)}
          </span>
        );
      })}
    </>
  );
}

const caption = "text-xs font-bold tracking-wider uppercase";

function Correction({ lines }: { lines: string[] }) {
  const [head, ...rest] = lines;
  const title = head.replace(/^\s*✏️?\s*/, "");

  return (
    <div className="bg-background border-border space-y-2 rounded-lg border p-3">
      <div className={cn(caption, "text-muted-foreground")}>✏️ {title}</div>
      {rest.map((line, i) => {
        const wrong = /^\s*(❌|✗|✘)\s*/.exec(line);
        const right = /^\s*(✅|✓|✔)\s*/.exec(line);
        const m = wrong ?? right;

        if (!m)
          return (
            <p key={i}>
              <Inline text={line} />
            </p>
          );

        return (
          <p
            key={i}
            className={cn(
              "flex gap-2 border-l-2 pl-3",
              wrong
                ? "border-destructive/60 line-through decoration-1"
                : "border-foreground/60 font-medium",
            )}
          >
            <span aria-hidden className="no-underline">
              {m[1]}
            </span>
            <span>
              <Inline text={line.slice(m[0].length)} />
            </span>
          </p>
        );
      })}
    </div>
  );
}

export function Labelled({
  label,
  text,
  trigger,
}: {
  label: string;
  text: string;
  trigger?: Trigger;
}) {
  const style = trigger?.style ?? "frame";
  const colors = palette[trigger?.color ?? "neutral"];

  // An inline style on a "Word:" paragraph marks the word, not a box.
  if (trigger && !blockStyles.includes(style)) {
    return (
      <p className="whitespace-pre-wrap">
        <Mark trigger={trigger}>{label}</Mark>: <Inline text={text} />
      </p>
    );
  }

  return (
    <div
      className={cn(
        "space-y-1.5",
        style === "bar"
          ? cn("border-l-4 py-0.5 pl-3", colors.bar)
          : cn("rounded-lg border p-3", colors.frame),
      )}
    >
      <div className={cn(caption, colors.label)}>{label}</div>
      {text && (
        <p className="whitespace-pre-wrap">
          <Inline text={text} />
        </p>
      )}
    </div>
  );
}

function Dialogue({ turns }: { turns: { speaker: string; text: string }[] }) {
  const first = turns[0].speaker.toLowerCase();

  return (
    <div className="space-y-1.5">
      {turns.map((turn, i) => {
        const right = turn.speaker.toLowerCase() !== first;

        return (
          <div
            key={i}
            className={cn("flex flex-col", right ? "items-end" : "items-start")}
          >
            <span className="text-muted-foreground px-1 text-[11px] font-semibold">
              {turn.speaker}
            </span>
            <p
              className={cn(
                "max-w-[85%] rounded-xl border px-3 py-1.5",
                right
                  ? "bg-secondary border-border rounded-br-sm"
                  : "bg-background border-border rounded-bl-sm",
              )}
            >
              <Inline text={turn.text} />
            </p>
          </div>
        );
      })}
    </div>
  );
}

export function MessageBody({ text }: { text: string }) {
  const settings = useTriggerSettings();
  const blocks = parseMessage(text, settings);

  return (
    <div className="space-y-3">
      {blocks.map((block, i) => {
        switch (block.type) {
          case "correction":
            return <Correction key={i} lines={block.lines} />;
          case "labelled":
            return <Labelled key={i} {...block} />;
          case "dialogue":
            return <Dialogue key={i} turns={block.turns} />;
          case "list":
            return (
              <ul key={i} className="space-y-1.5">
                {block.items.map((item, n) => (
                  <li key={n} className="flex gap-3">
                    <span
                      aria-hidden
                      className="text-muted-foreground w-4 shrink-0 text-[13px] font-semibold"
                    >
                      {block.ordered ? n + 1 : "•"}
                    </span>
                    <span className="min-w-0">
                      {block.ordered ? (
                        <Exercise text={item} />
                      ) : (
                        <Inline text={item} />
                      )}
                    </span>
                  </li>
                ))}
              </ul>
            );
          default:
            return (
              <p key={i} className="whitespace-pre-wrap">
                <Inline text={block.text} />
              </p>
            );
        }
      })}
    </div>
  );
}
