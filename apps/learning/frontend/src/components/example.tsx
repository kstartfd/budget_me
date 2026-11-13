import { Fragment } from "react"
import { cn } from "@/lib/utils"

/**
 * An example inside a running text is written between backticks, by the
 * teacher's texts in the library and by every model that writes for the
 * learner. The page sets it in a font of its own, on the readability sheet's
 * blue wash, so it is seen at once which words are an example and which are
 * the explanation. The server's prompts ask for this mark (`exampleNote`).
 */
export const examplePattern = /(`[^`\n]+`)/g

/** The look of an example inside a line of text. */
export const exampleClass =
  "bg-lang text-lang-foreground rounded-[4px] px-[4px] py-px font-mono text-[0.9em] [box-decoration-break:clone]"

/** The look of an example that stands on a line of its own (a list of sentences). */
export const exampleLineClass = "font-mono"

export function Example({ children, className }: { children: React.ReactNode; className?: string }) {
  return <span className={cn(exampleClass, className)}>{children}</span>
}

/** Splits a text at its examples: each one is set in the example look, the rest goes through `plain`. */
export function withExamples(text: string, plain: (part: string) => React.ReactNode): React.ReactNode[] {
  return text.split(examplePattern).map((part, index) =>
    index % 2 === 1 ? (
      <Example key={index}>{part.slice(1, -1)}</Example>
    ) : (
      <Fragment key={index}>{plain(part)}</Fragment>
    ),
  )
}
