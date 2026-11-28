"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { IconArrowLeft } from "@tabler/icons-react";
import { LocaleSwitch, LogoutButton, ThemeButton } from "@/components/header-controls";

import { AiSection } from "@/features/ai";
import { ReminderSection } from "@/features/learning/components/reminder-section";
import { TriggersSection } from "@/features/chat/components/triggers-section";

import { AccountSection } from "./account-section";
import { DeleteAccountSection } from "./delete-account-section";
import { ExtensionSection } from "./extension-section";

export function SettingsScreen() {
  const t = useTranslations("learning");
  const workspace = useTranslations("workspace");

  return (
    <div className="anim-fade mx-auto flex w-full max-w-[705px] flex-col gap-7 lg:pt-6">
      <header className="flex flex-col gap-3.5">
        <Link
          href="/learning/overview"
          className="text-foreground/80 flex w-fit items-center gap-1 text-xs"
        >
          <IconArrowLeft size={14} aria-hidden="true" />
          {t("title")}
        </Link>
        <h1 className="text-[26px] leading-tight font-semibold tracking-[-0.02em]">
          {t("settings")}
        </h1>
      </header>

      <section className="bg-card border-border flex flex-wrap items-center justify-between gap-3 rounded-lg border p-5">
        <div>
          <h2 className="text-sm font-semibold">{workspace("languagesTitle")}</h2>
          <p className="text-muted-foreground mt-1 text-xs">{workspace("languagesDescription")}</p>
        </div>
        <Link href="/learning/languages" className="border-border hover:bg-muted rounded-md border px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-ring">
          {workspace("manageLanguages")}
        </Link>
      </section>

      <section className="bg-card border-border rounded-lg border p-5">
        <h2 className="text-sm font-semibold">{workspace("interfaceSettings")}</h2>
        <p className="text-muted-foreground mt-1 text-xs">{workspace("interfaceDescription")}</p>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <LocaleSwitch className="w-[200px]" />
          <ThemeButton />
          <LogoutButton />
        </div>
      </section>

      <AccountSection />
      <AiSection />
      <ReminderSection />
      <TriggersSection />
      <ExtensionSection />
      <DeleteAccountSection />
    </div>
  );
}
