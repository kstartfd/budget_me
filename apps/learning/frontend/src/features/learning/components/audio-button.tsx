"use client"

import { useEffect, useRef, useState } from "react"
import { useTranslations } from "next-intl"
import { IconLoader2, IconVolume } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { fetchEntryAudio } from "../api"

type PlaybackState = "idle" | "loading" | "playing" | "error"

type ActivePlayback = {
  key: string
  audio: HTMLAudioElement
}

let activePlayback: ActivePlayback | null = null
const audioUrls = new Map<string, string>()

async function audioUrlFor(entryId: string): Promise<string> {
  const cached = audioUrls.get(entryId)
  if (cached) return cached

  const blob = await fetchEntryAudio(entryId)
  const url = URL.createObjectURL(blob)

  if (audioUrls.size >= 50) {
    const oldest = audioUrls.entries().next().value
    if (oldest) {
      audioUrls.delete(oldest[0])
      URL.revokeObjectURL(oldest[1])
    }
  }

  audioUrls.set(entryId, url)
  return url
}

function stopActivePlayback(exceptKey?: string) {
  if (!activePlayback || activePlayback.key === exceptKey) return

  activePlayback.audio.pause()
  activePlayback.audio.currentTime = 0
  activePlayback = null
}

export function EntryAudioButton({
  entryId,
  compact = false,
}: {
  entryId: string
  compact?: boolean
}) {
  const t = useTranslations("learning")
  const [state, setState] = useState<PlaybackState>("idle")
  const audioRef = useRef<HTMLAudioElement | null>(null)

  useEffect(() => {
    const audio = audioRef.current

    return () => {
      if (activePlayback?.audio === audio) {
        audio.pause()
        activePlayback = null
      }
    }
  }, [])

  const toggle = async () => {
    if (state === "loading") return

    if (state === "playing" && audioRef.current) {
      audioRef.current.pause()
      audioRef.current.currentTime = 0
      if (activePlayback?.audio === audioRef.current) activePlayback = null
      setState("idle")
      return
    }

    setState("loading")

    try {
      const url = await audioUrlFor(entryId)
      let audio = audioRef.current

      if (!audio) {
        audio = new Audio()
        audio.addEventListener("ended", () => {
          if (activePlayback?.audio === audio) activePlayback = null
          setState("idle")
        })
        audio.addEventListener("error", () => setState("error"))
        audioRef.current = audio
      }

      stopActivePlayback(entryId)
      audio.src = url
      activePlayback = { key: entryId, audio }
      await audio.play()
      setState("playing")
    } catch {
      setState("error")
    }
  }

  return (
    <Button
      type="button"
      variant="secondary"
      size={compact ? "icon-sm" : "icon"}
      aria-label={t("playAudio")}
      title={state === "error" ? t("audioError") : t("playAudio")}
      aria-pressed={state === "playing"}
      disabled={state === "loading"}
      onClick={(event) => {
        event.stopPropagation()
        void toggle()
      }}
      onKeyDown={(event) => event.stopPropagation()}
      className={cn(
        "text-muted-foreground hover:text-primary rounded-full",
        compact ? "size-8 shrink-0" : "size-10",
        state === "playing" && "text-primary bg-primary/10",
        state === "error" && "text-destructive bg-destructive/10",
      )}
    >
      {state === "loading" ? (
        <IconLoader2 size={compact ? 15 : 18} className="animate-spin" />
      ) : (
        <IconVolume size={compact ? 16 : 19} />
      )}
      <span className="sr-only">{t("playAudio")}</span>
    </Button>
  )
}
