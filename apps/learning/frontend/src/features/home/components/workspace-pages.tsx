"use client";

import { useState } from "react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { IconArrowRight, IconBook2, IconBolt, IconRefresh } from "@tabler/icons-react";

import { Button } from "@/components/ui/button";
import { useLoopDashboard, usePath } from "@/features/path/hooks";
import { RulesIndex } from "@/features/path/components/rule-book";
import type { LoopActivity, PathTeacher } from "@/features/path/schemas";
import { cn } from "@/lib/utils";

import { useActiveTeacher } from "../setup";
import { card } from "./parts";
import { ReviewActivity } from "./review-activity";
import { useLanguageName } from "./language-switcher";

function Heading({ title, description }: { title: string; description: string }) {
  const t = useTranslations("workspace");
  return <header className="space-y-1">
    <p className="text-muted-foreground text-xs">{t("crumb", { page: title })}</p>
    <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
    <p className="text-muted-foreground max-w-[68ch] text-sm">{description}</p>
  </header>;
}

function EmptyLanguage() {
  const t = useTranslations("workspace");
  return <section className={cn(card, "space-y-3 p-6")}>
    <p className="text-sm">{t("noLanguage")}</p>
    <Button asChild><Link href="/learning/start">{t("addLanguage")}</Link></Button>
  </section>;
}

export function KnowledgeWorkspace() {
  const t = useTranslations("workspace");
  const { teachers, active } = useActiveTeacher();
  return <div className="anim-fade space-y-5">
    <Heading title={t("knowledgeTitle")} description={t("knowledgeDescription")} />
    <section className={cn(card, "flex flex-wrap items-center justify-between gap-4 p-4")}>
      <div>
        <h2 className="text-sm font-semibold">{t("notebookTitle")}</h2>
        <p className="text-muted-foreground mt-1 max-w-[65ch] text-xs">{t("notebookDescription")}</p>
      </div>
      <Button asChild variant="outline" size="sm"><Link href="/learning">{t("openNotebook")}<IconArrowRight size={14} aria-hidden="true" /></Link></Button>
    </section>
    {teachers.isPending ? <p className="text-muted-foreground text-sm">{t("loading")}</p>
      : teachers.isError ? <ErrorState message={teachers.error.message} onRetry={() => void teachers.refetch()} />
      : active ? <RuleBook language={active.language} /> : <EmptyLanguage />}
  </div>;
}

// The Rule Book remains the existing read-only content API. Free rule practice
// lives on a rule detail page; it is not counted as mastery evidence.
function RuleBook({ language }: { language: string }) {
  const t = useTranslations("workspace");
  const name = useLanguageName();
  return <section className="space-y-3" aria-label={t("ruleBookTitle")}>
    <div className="flex items-center gap-2 text-sm font-semibold"><IconBook2 size={17} aria-hidden="true" />{t("ruleBookTitle", { language: name(language) })}</div>
    <RuleBookContents key={language} language={language} />
  </section>;
}

function RuleBookContents({ language }: { language: string }) {
  // The language prop is explicit, so switching languages does not reuse old rules.
  return <RulesIndex language={language} />;
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  const t = useTranslations("workspace");
  return <section role="alert" className={cn(card, "space-y-3 p-5")}>
    <p className="text-destructive text-sm">{message}</p>
    <Button variant="outline" onClick={onRetry}>{t("retry")}</Button>
  </section>;
}

export function PracticeWorkspace() {
  const t = useTranslations("workspace");
  const { teachers, active } = useActiveTeacher();
  return <div className="anim-fade space-y-5">
    <Heading title={t("practiceTitle")} description={t("practiceDescription")} />
    {teachers.isPending ? <p className="text-muted-foreground text-sm">{t("loading")}</p>
      : teachers.isError ? <ErrorState message={teachers.error.message} onRetry={() => void teachers.refetch()} />
      : active ? <PracticeForLanguage key={active.id} teacher={active} /> : <EmptyLanguage />}
  </div>;
}

function PracticeForLanguage({ teacher }: { teacher: PathTeacher }) {
  const t = useTranslations("workspace");
  const name = useLanguageName();
  const loop = useLoopDashboard(teacher.language);
  const [selected, setSelected] = useState<LoopActivity | null>(null);
  const due = (loop.data?.activities ?? []).filter((activity) => activity.type === "review" && activity.exercise);
  const pathHref = `/learning/path?language=${encodeURIComponent(teacher.language)}`;
  const bookHref = `/learning/knowledge?language=${encodeURIComponent(teacher.language)}`;

  return <>
    {selected && <ReviewActivity key={selected.id} activity={selected} onClose={() => setSelected(null)} />}
    <p className="text-muted-foreground text-xs">{t("selectedLanguage", { language: name(teacher.language) })}</p>
    <div className="grid gap-4 md:grid-cols-2">
      <section className={cn(card, "space-y-3 p-5")}>
        <IconRefresh size={18} className="text-muted-foreground" aria-hidden="true" />
        <h2 className="font-semibold">{t("dueReviewTitle")}</h2>
        {loop.isPending ? <p className="text-muted-foreground text-sm">{t("loading")}</p>
          : loop.isError ? <ErrorState message={loop.error.message} onRetry={() => void loop.refetch()} />
          : due.length === 0 ? <p className="text-muted-foreground text-sm">{t("nothingDue")}</p>
          : <ul className="space-y-2">{due.map((activity) => <li key={activity.id} className="border-border flex items-center justify-between gap-3 border-t py-2">
            <span className="min-w-0 truncate text-sm">{activity.title}</span>
            <Button variant="outline" size="sm" onClick={() => setSelected(activity)}>{t("startReview")}</Button>
          </li>)}</ul>}
      </section>
      <section className={cn(card, "space-y-3 p-5")}>
        <IconBolt size={18} className="text-muted-foreground" aria-hidden="true" />
        <h2 className="font-semibold">{t("trackedPracticeTitle")}</h2>
        <p className="text-muted-foreground text-sm">{teacher.path ? t("trackedPracticeDescription") : t("noProgramPractice")}</p>
        <Button asChild variant="outline"><Link href={teacher.path ? pathHref : "/learning/overview"}>{teacher.path ? t("openCurrentActivity") : t("chooseStartingPoint")}</Link></Button>
      </section>
      <section className={cn(card, "space-y-3 p-5 md:col-span-2")}>
        <h2 className="font-semibold">{t("freePracticeTitle")}</h2>
        <p className="text-muted-foreground text-sm">{t("freePracticeDescription")}</p>
        <div className="flex flex-wrap gap-2">
          <Button asChild variant="outline"><Link href={bookHref}>{t("openRuleBook")}</Link></Button>
          <Button asChild variant="outline"><Link href="/learning">{t("openNotebook")}</Link></Button>
        </div>
      </section>
    </div>
  </>;
}

export function ProgressWorkspace() {
  const t = useTranslations("workspace");
  const { teachers, active } = useActiveTeacher();
  return <div className="anim-fade space-y-5">
    <Heading title={t("progressTitle")} description={t("progressDescription")} />
    {teachers.isPending ? <p className="text-muted-foreground text-sm">{t("loading")}</p>
      : teachers.isError ? <ErrorState message={teachers.error.message} onRetry={() => void teachers.refetch()} />
      : active ? <ProgressForLanguage key={active.id} teacher={active} /> : <EmptyLanguage />}
  </div>;
}

function ProgressForLanguage({ teacher }: { teacher: PathTeacher }) {
  const t = useTranslations("workspace");
  const name = useLanguageName();
  const loop = useLoopDashboard(teacher.language);
  const path = usePath(teacher.language, teacher.translationLanguage).active;
  const topics = path.data?.topics ?? [];
  const completed = topics.filter((topic) => topic.status === "completed").length;
  return <>
    <p className="text-muted-foreground text-xs">{t("selectedLanguage", { language: name(teacher.language) })}</p>
    {path.isError ? <ErrorState message={path.error.message} onRetry={() => void path.refetch()} /> : null}
    {loop.isError ? <ErrorState message={loop.error.message} onRetry={() => void loop.refetch()} /> : null}
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <Metric label={t("programTopics")} value={path.data && topics.length > 0 ? `${completed} / ${topics.length}` : null} />
      <Metric label={t("trackedItems")} value={loop.data?.items.tracked} />
      <Metric label={t("masteredItems")} value={loop.data?.items.mastered} />
      <Metric label={t("dueReviews")} value={loop.data?.reviews.due} />
    </div>
    <section className={cn(card, "space-y-3 p-5")}>
      <h2 className="font-semibold">{t("evidenceTitle")}</h2>
      {loop.data ? <p className="text-sm">{t("evidenceSummary", { total: loop.data.evidence.total, recent: loop.data.evidence.last7Days })}</p>
        : <p className="text-muted-foreground text-sm">{t("insufficientData")}</p>}
      <p className="text-muted-foreground text-xs">{t("estimatesUnavailable")}</p>
    </section>
    <Button asChild variant="outline"><Link href={`/learning/path?language=${encodeURIComponent(teacher.language)}`}>{t("openPath")}</Link></Button>
  </>;
}

function Metric({ label, value }: { label: string; value: number | string | null | undefined }) {
  const t = useTranslations("workspace");
  return <section className={cn(card, "space-y-2 p-4")}>
    <p className="text-muted-foreground text-xs">{label}</p>
    <p className="text-2xl font-semibold">{value ?? "–"}</p>
    {value == null && <p className="text-muted-foreground text-xs">{t("insufficientData")}</p>}
  </section>;
}
