const emphasisPattern = /(\*\*\*[^*]+\*\*\*|\*\*[^*]+\*\*|\*[^*]+\*)/g

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
          return <em key={index}>{part.slice(2, -2)}</em>
        }
        if (part.startsWith("*") && part.endsWith("*") && part.length > 2) {
          return <strong key={index}>{part.slice(1, -1)}</strong>
        }
        return part
      })}
    </>
  )
}
