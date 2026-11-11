import {
  IconAbc,
  IconBook,
  IconHeadphones,
  IconMicrophone,
  IconPencil,
  IconSchool,
} from "@tabler/icons-react"
import { cn } from "@/lib/utils"

/** The skills a program topic can teach, in the order they are listed. */
export const skillOrder = [
  "grammar",
  "vocabulary",
  "reading",
  "listening",
  "speaking",
  "writing",
] as const

export type Skill = (typeof skillOrder)[number]

/** An icon and a colour for each skill, so a program reads at a glance. */
const meta: Record<Skill, { icon: typeof IconSchool; text: string; bar: string }> = {
  grammar: { icon: IconSchool, text: "text-sky-600 dark:text-sky-400", bar: "bg-sky-500" },
  vocabulary: { icon: IconAbc, text: "text-violet-600 dark:text-violet-400", bar: "bg-violet-500" },
  reading: { icon: IconBook, text: "text-amber-600 dark:text-amber-400", bar: "bg-amber-500" },
  listening: { icon: IconHeadphones, text: "text-teal-600 dark:text-teal-400", bar: "bg-teal-500" },
  speaking: { icon: IconMicrophone, text: "text-rose-600 dark:text-rose-400", bar: "bg-rose-500" },
  writing: { icon: IconPencil, text: "text-emerald-600 dark:text-emerald-400", bar: "bg-emerald-500" },
}

const known = (skill: string): skill is Skill => skill in meta

export function skillBar(skill: string): string {
  return known(skill) ? meta[skill].bar : "bg-muted-foreground"
}

/** The icon of a skill; an unknown skill from an older program gets none. */
export function SkillIcon({
  skill,
  size = 14,
  className,
}: {
  skill: string
  size?: number
  className?: string
}) {
  if (!known(skill)) return null
  const { icon: Icon, text } = meta[skill]

  return <Icon size={size} aria-hidden="true" className={cn("shrink-0", text, className)} />
}
