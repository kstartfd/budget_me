"use client"

import { useEffect, useRef, useState } from "react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import {
  IconCamera,
  IconCheck,
  IconLoader2,
  IconPhoto,
  IconSettings,
} from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { useAiSettings } from "@/features/ai"
import { LearningModal } from "@/features/learning/components/learning-modal"

import { ScanError, scanErrorCode } from "../api"
import { useAddFound, useScanPicture } from "../hooks"
import type { Found } from "../schemas"

/**
 * Words and phrases from a photo: the learner takes or uploads a picture, the
 * backend has their AI agent read it, and they keep what they want. The kept
 * items are saved as captures, so the agent fills in their cards like it does
 * for the browser extension.
 */
export function ScanModal({
  topicId,
  onClose,
}: {
  topicId: string
  onClose: () => void
}) {
  const t = useTranslations("scan")
  const learning = useTranslations("learning")
  const agent = useAiSettings()
  const scan = useScanPicture(topicId)
  const add = useAddFound(topicId, t("source"))
  const camera = useRef<HTMLInputElement>(null)
  const gallery = useRef<HTMLInputElement>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [items, setItems] = useState<Found[]>([])
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [added, setAdded] = useState<number | null>(null)
  const [notImage, setNotImage] = useState(false)

  // The preview URL is revoked when it is replaced and when the modal goes.
  useEffect(() => {
    return () => {
      if (preview) URL.revokeObjectURL(preview)
    }
  }, [preview])

  const noAgent = agent.isSuccess && agent.data === null
  const reading = scan.isPending
  const chosen = scan.isSuccess && !reading

  const pick = (file: File | undefined) => {
    if (!file) return

    if (!file.type.startsWith("image/")) {
      setNotImage(true)

      return
    }

    setNotImage(false)
    setPreview(URL.createObjectURL(file))
    add.reset()
    scan.mutate(file, {
      onSuccess: (found) => {
        setItems(found)
        setSelected(new Set(found.keys()))
      },
    })
  }

  const again = () => {
    scan.reset()
    add.reset()
    setItems([])
    setSelected(new Set())
    setAdded(null)
    setPreview(null)
  }

  const toggle = (index: number) =>
    setSelected((current) => {
      const next = new Set(current)
      if (!next.delete(index)) next.add(index)

      return next
    })

  const submit = () => {
    const wanted = items.filter((_, i) => selected.has(i))
    if (wanted.length === 0) return

    add.mutate(wanted, {
      onSuccess: ({ added: count, failed }) => {
        // What went through is saved; what did not stays for another try, and
        // with nothing left the list gives way to the result.
        setAdded((added ?? 0) + count)
        setItems(failed)
        setSelected(new Set(failed.keys()))
      },
    })
  }

  const scanMessage = (e: unknown) => {
    const code = e instanceof ScanError ? e.code : undefined

    switch (code) {
      case scanErrorCode.invalid:
        return (e as ScanError).message
      case scanErrorCode.rejected:
        return t("errors.rejected")
      case scanErrorCode.unavailable:
        return t("errors.unavailable")
      case scanErrorCode.agentFailed:
        return t("errors.agentFailed")
      case scanErrorCode.tooMany:
        return t("errors.tooMany")
      default:
        return t("errors.failed")
    }
  }

  const choose = (input: React.RefObject<HTMLInputElement | null>) => (
    <input
      ref={input}
      type="file"
      accept="image/*"
      capture={input === camera ? "environment" : undefined}
      className="sr-only"
      tabIndex={-1}
      aria-hidden
      onChange={(e) => {
        pick(e.target.files?.[0])
        // The same file can be picked again after a failed read.
        e.target.value = ""
      }}
    />
  )

  return (
    <LearningModal title={t("title")} onClose={onClose}>
      {choose(camera)}
      {choose(gallery)}

      {added !== null && items.length === 0 ? (
        <div className="space-y-4">
          <p className="flex items-center gap-2 text-sm font-semibold">
            <IconCheck size={18} className="text-success" />
            {t("added", { count: added })}
          </p>
          <p className="text-muted-foreground text-sm leading-relaxed">
            {t("addedHint")}
          </p>
          <div className="flex gap-3">
            <Button variant="secondary" className="flex-1" onClick={again}>
              {t("another")}
            </Button>
            <Button className="flex-1" onClick={onClose}>
              {t("close")}
            </Button>
          </div>
        </div>
      ) : reading ? (
        <div className="space-y-4">
          {preview && (
            // eslint-disable-next-line @next/next/no-img-element -- a local blob URL
            <img
              src={preview}
              alt={t("previewAlt")}
              className="bg-secondary max-h-56 w-full rounded-2xl object-contain"
            />
          )}
          <p
            role="status"
            className="text-muted-foreground flex items-center justify-center gap-2 text-sm"
          >
            <IconLoader2 size={16} className="animate-spin" />
            {t("reading")}
          </p>
        </div>
      ) : chosen ? (
        <div className="space-y-3">
          {items.length === 0 ? (
            <>
              <p className="text-muted-foreground text-sm leading-relaxed">
                {t("nothing")}
              </p>
              <Button variant="secondary" className="w-full" onClick={again}>
                {t("retry")}
              </Button>
            </>
          ) : (
            <>
              <div className="flex items-center justify-between gap-3">
                <p className="text-sm font-semibold">
                  {t("found", { count: items.length })}
                </p>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-primary h-auto px-0 py-0 text-xs"
                  onClick={() =>
                    setSelected(
                      selected.size === items.length
                        ? new Set()
                        : new Set(items.keys()),
                    )
                  }
                >
                  {t(
                    selected.size === items.length
                      ? "selectNone"
                      : "selectAll",
                  )}
                </Button>
              </div>
              <ul className="space-y-1.5">
                {items.map((item, index) => (
                  <li key={`${item.kind}:${item.text}`}>
                    <label className="bg-secondary flex cursor-pointer items-start gap-3 rounded-2xl px-3.5 py-3">
                      <Checkbox
                        checked={selected.has(index)}
                        onCheckedChange={() => toggle(index)}
                        className="mt-0.5"
                      />
                      <span className="min-w-0 flex-1">
                        <span className="block text-sm leading-snug font-semibold break-words">
                          {item.text}
                        </span>
                        {item.translation && (
                          <span className="text-muted-foreground mt-0.5 block text-xs leading-snug break-words">
                            {item.translation}
                          </span>
                        )}
                      </span>
                      <span className="text-faint shrink-0 text-[11px]">
                        {learning(item.kind)}
                      </span>
                    </label>
                  </li>
                ))}
              </ul>
              {add.isError || (add.data && add.data.failed.length > 0) ? (
                <p role="alert" className="text-destructive text-sm">
                  {t("addFailed")}
                </p>
              ) : null}
              <div className="bg-card sticky bottom-0 -mx-1 flex gap-3 px-1 pt-2">
                <Button variant="secondary" onClick={again}>
                  {t("retry")}
                </Button>
                <Button
                  className="flex-1"
                  disabled={add.isPending || selected.size === 0}
                  onClick={submit}
                >
                  {add.isPending
                    ? t("adding")
                    : t("add", { count: selected.size })}
                </Button>
              </div>
            </>
          )}
        </div>
      ) : (
        <div className="space-y-4">
          <p className="text-muted-foreground text-sm leading-relaxed">
            {t("hint")}
          </p>
          {noAgent ? (
            <div className="bg-secondary space-y-3 rounded-2xl p-4">
              <p className="text-sm leading-relaxed">{t("needAgent")}</p>
              <Button asChild variant="secondary" className="text-primary">
                <Link href="/learning/settings">
                  <IconSettings size={16} />
                  {t("openSettings")}
                </Link>
              </Button>
            </div>
          ) : (
            <div className="grid gap-2.5">
              <Button
                className="h-12 rounded-2xl text-[15px] font-semibold"
                onClick={() => camera.current?.click()}
              >
                <IconCamera size={18} />
                {t("takePhoto")}
              </Button>
              <Button
                variant="secondary"
                className="h-12 rounded-2xl text-[15px] font-semibold"
                onClick={() => gallery.current?.click()}
              >
                <IconPhoto size={18} />
                {t("choosePhoto")}
              </Button>
            </div>
          )}
          {notImage && (
            <p role="alert" className="text-destructive text-sm">
              {t("notImage")}
            </p>
          )}
          {scan.isError && (
            <div role="alert" className="space-y-2">
              <p className="text-destructive text-sm">
                {scanMessage(scan.error)}
              </p>
              {scan.error instanceof ScanError &&
                scan.error.code === scanErrorCode.rejected && (
                  <Link
                    href="/learning/settings"
                    className="text-primary text-sm"
                  >
                    {t("openSettings")}
                  </Link>
                )}
            </div>
          )}
        </div>
      )}
    </LearningModal>
  )
}
