"use client";

import { cn } from "@/lib/utils";
import type { Entry } from "../schemas";
import { EntryAudioButton } from "./audio-button";

/**
 * A flashcard that turns over. One side shows the word or phrase in the language
 * being learned (with its transcription and audio), the other its translation
 * with the breakdown and an example; `front` says which side is face up.
 * Tapping the card calls `onFlip`. Mount it with a `key` per card, so the next
 * card appears face up at once instead of turning from the old answer.
 */
export function FlipCard({
  entry,
  front,
  flipped,
  onFlip,
  hint,
}: {
  entry: Entry;
  front: "text" | "translation";
  flipped: boolean;
  onFlip: () => void;
  hint: string;
}) {
  const translation = entry.translation || entry.text;

  const learned = (
    <>
      <p className="text-2xl font-semibold break-words">{entry.text}</p>
      {entry.transcription && (
        <p className="text-muted-foreground font-mono text-sm">
          {entry.transcription}
        </p>
      )}
      <div onClick={(e) => e.stopPropagation()}>
        <EntryAudioButton entryId={entry.id} />
      </div>
    </>
  );

  const details = (
    <>
      {entry.explanation && (
        <p className="text-muted-foreground leading-relaxed">
          {entry.explanation}
        </p>
      )}
      {entry.example && (
        <p className="leading-relaxed italic">{entry.example}</p>
      )}
    </>
  );

  const textFirst = front === "text";

  return (
    <div onClick={onFlip} className="cursor-pointer [perspective:1200px]">
      <div
        className={cn(
          "grid [transform-style:preserve-3d] transition-transform duration-500 ease-out motion-reduce:transition-none",
          flipped && "[transform:rotateY(180deg)]",
        )}
      >
        <div
          aria-hidden={flipped}
          className="border-border bg-background col-start-1 row-start-1 flex min-h-48 flex-col items-center justify-center gap-3 rounded-lg border p-5 text-center [backface-visibility:hidden]"
        >
          {textFirst ? (
            learned
          ) : (
            <p className="text-2xl font-semibold break-words">{translation}</p>
          )}
          <p className="text-faint text-xs">{hint}</p>
        </div>

        <div
          aria-hidden={!flipped}
          className="border-border bg-muted/40 col-start-1 row-start-1 flex min-h-48 flex-col items-center justify-center gap-3 rounded-lg border p-5 text-center text-sm [backface-visibility:hidden] [transform:rotateY(180deg)]"
        >
          {textFirst ? (
            <>
              <p className="text-xl font-semibold break-words">{translation}</p>
              <div className="space-y-2 text-left">{details}</div>
            </>
          ) : (
            <>
              {learned}
              <div className="space-y-2 text-left">{details}</div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
