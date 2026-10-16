/**
 * Copies text to the clipboard. Telegram's webviews and Safari reject
 * `writeText` once the click handler has awaited something (a secret still
 * being fetched), and some webviews have no Clipboard API at all, so try in
 * order: the async API handed a promise (keeps the click's user gesture alive),
 * plain `writeText`, then `execCommand("copy")` on a hidden textarea.
 *
 * Resolves false when every route is refused; rejects only if `text` rejects.
 */
export async function copyToClipboard(
  text: string | Promise<string>,
): Promise<boolean> {
  if (typeof ClipboardItem !== "undefined" && navigator.clipboard?.write) {
    try {
      const blob = Promise.resolve(text).then(
        (value) => new Blob([value], { type: "text/plain" }),
      )
      await navigator.clipboard.write([
        new ClipboardItem({ "text/plain": blob }),
      ])
      return true
    } catch {
      // Refused or unsupported: fall back to the older routes below.
    }
  }

  const value = await text
  try {
    await navigator.clipboard.writeText(value)
    return true
  } catch {
    return legacyCopy(value)
  }
}

function legacyCopy(value: string): boolean {
  const area = document.createElement("textarea")
  area.value = value
  area.setAttribute("readonly", "")
  area.style.cssText = "position:fixed;top:0;left:0;opacity:0"
  document.body.appendChild(area)
  area.select()
  area.setSelectionRange(0, value.length)
  try {
    return document.execCommand("copy")
  } catch {
    return false
  } finally {
    area.remove()
  }
}
