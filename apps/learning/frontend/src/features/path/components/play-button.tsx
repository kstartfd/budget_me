"use client"

import { useEffect, useRef, useState } from "react"
import { useTranslations } from "next-intl"
import { IconLoader2, IconPlayerStopFilled, IconVolume } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { fetchPathAudio } from "../api"

type State = "idle" | "loading" | "playing" | "error"

// The sound of a text is made once on the server; here it is kept for the
// session so hearing it again costs nothing.
const urls = new Map<string, string>()

/**
 * Plays a voice from the program: a word, a sentence to listen to, a phrase to
 * repeat. Only one plays at a time.
 */
let current: HTMLAudioElement | null = null

export function PlayButton({
  path,
  label,
  compact = false,
}: {
  /** The audio endpoint, relative to the learning API. */
  path: string
  label: string
  compact?: boolean
}) {
  const t = useTranslations("path.topic")
  const [state, setState] = useState<State>("idle")
  const audio = useRef<HTMLAudioElement | null>(null)

  useEffect(() => {
    return () => {
      audio.current?.pause()
    }
  }, [])

  const play = async () => {
    if (state === "playing") {
      audio.current?.pause()
      setState("idle")

      return
    }

    try {
      setState("loading")
      let url = urls.get(path)
      if (!url) {
        url = URL.createObjectURL(await fetchPathAudio(path))
        urls.set(path, url)
      }

      current?.pause()
      const element = new Audio(url)
      audio.current = element
      current = element
      element.onended = () => setState("idle")
      element.onerror = () => setState("error")
      await element.play()
      setState("playing")
    } catch {
      setState("error")
    }
  }

  const Icon = state === "loading" ? IconLoader2 : state === "playing" ? IconPlayerStopFilled : IconVolume

  return (
    <Button
      type="button"
      variant={compact ? "ghost" : "outline"}
      size={compact ? "icon" : "lg"}
      onClick={() => void play()}
      aria-label={label}
      title={state === "error" ? t("audioError") : label}
      className={cn(compact ? "size-8" : "text-[13px]", state === "error" && "text-destructive")}
    >
      <Icon size={compact ? 15 : 16} className={state === "loading" ? "animate-spin" : undefined} />
      {!compact && label}
    </Button>
  )
}
