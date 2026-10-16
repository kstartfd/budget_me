"use client"

import { useEffect, useState } from "react"
import { useTranslations } from "next-intl"
import { IconCheck, IconCopy, IconEye, IconEyeOff } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { copyToClipboard } from "@/lib/clipboard"

type Feedback = "copied" | "copyError" | "loadError" | null

/**
 * A secret kept masked until the eye is pressed, with a copy button beside it.
 * `load` supplies the value on demand (a list row only knows the token's id)
 * and the value is dropped again on hide, so a masked field never holds it.
 * If the clipboard refuses, the secret is shown so it can be selected by hand.
 */
export function SecretField({
  mask,
  load,
  compact = false,
}: {
  mask: string
  load: () => Promise<string>
  compact?: boolean
}) {
  const t = useTranslations("learning")
  const [secret, setSecret] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [feedback, setFeedback] = useState<Feedback>(null)

  useEffect(() => {
    if (feedback !== "copied") return
    const timer = setTimeout(() => setFeedback(null), 2000)
    return () => clearTimeout(timer)
  }, [feedback])

  async function toggle() {
    if (busy) return
    setFeedback(null)
    if (secret !== null) {
      setSecret(null)
      return
    }
    setBusy(true)
    try {
      setSecret(await load())
    } catch {
      setFeedback("loadError")
    } finally {
      setBusy(false)
    }
  }

  async function copy() {
    if (busy) return
    setFeedback(null)
    setBusy(true)
    // Started inside the click so the copy keeps the user gesture while the
    // secret is still being fetched.
    const pending = load()
    try {
      if (await copyToClipboard(pending)) {
        setFeedback("copied")
      } else {
        setSecret(await pending)
        setFeedback("copyError")
      }
    } catch {
      setFeedback("loadError")
    } finally {
      setBusy(false)
    }
  }

  const iconSize = compact ? 17 : 18
  const buttonClass = `${compact ? "h-11 w-11 rounded-xl" : "h-[52px] w-[52px] rounded-2xl"} shrink-0 aria-busy:opacity-60`

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <code
          className={`bg-secondary text-foreground min-w-0 flex-1 font-mono text-xs leading-relaxed select-text ${secret === null ? "overflow-hidden whitespace-nowrap" : "break-all"} ${compact ? "rounded-xl px-3.5 py-3" : "rounded-2xl px-4 py-3.5"}`}
        >
          {secret ?? mask}
        </code>
        <Button
          type="button"
          variant="secondary"
          size="icon"
          aria-label={secret === null ? t("showToken") : t("hideToken")}
          aria-busy={busy}
          className={buttonClass}
          onClick={() => void toggle()}
        >
          {secret === null ? (
            <IconEye size={iconSize} />
          ) : (
            <IconEyeOff size={iconSize} />
          )}
        </Button>
        <Button
          type="button"
          variant="secondary"
          size="icon"
          aria-label={t("copyToken")}
          aria-busy={busy}
          className={buttonClass}
          onClick={() => void copy()}
        >
          {feedback === "copied" ? (
            <IconCheck size={iconSize} />
          ) : (
            <IconCopy size={iconSize} />
          )}
        </Button>
      </div>
      <span role="status" className="sr-only">
        {feedback === "copied" ? t("copied") : ""}
      </span>
      {(feedback === "copyError" || feedback === "loadError") && (
        <p role="alert" className="text-destructive text-xs leading-relaxed">
          {t(feedback === "copyError" ? "copyError" : "tokenLoadError")}
        </p>
      )}
    </div>
  )
}
