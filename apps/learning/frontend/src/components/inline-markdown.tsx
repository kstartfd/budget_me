// Assistant replies can wrap emphasis across lines. Keep this deliberately
// small (the chat does not need a full Markdown renderer), but allow that.
const emphasisPattern = /(\*\*\*[\s\S]+?\*\*\*|\*\*[\s\S]+?\*\*|\*[^*\n]+?\*)/g

export function InlineMarkdown({ text }: { text: string }) {
  return (
    <>
      {text.split(emphasisPattern).map((part, index) => {
        if (part.startsWith("***") && part.endsWith("***") && part.length > 6) {
          return (
            <strong key={index}>
              <em>{part.slice(3, -3)}</em>
            </strong>
          )
        }
        if (part.startsWith("**") && part.endsWith("**") && part.length > 4) {
          return <strong key={index}>{part.slice(2, -2)}</strong>
        }
        if (part.startsWith("*") && part.endsWith("*") && part.length > 2) {
          return <em key={index}>{part.slice(1, -1)}</em>
        }
        return part
      })}
    </>
  )
}
