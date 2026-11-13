"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { IconArrowLeft } from "@tabler/icons-react";

import { AiSection } from "@/features/ai";
import { TriggersSection } from "@/features/chat/components/triggers-section";

import { AccountSection } from "./account-section";
import { DeleteAccountSection } from "./delete-account-section";
import { ExtensionSection } from "./extension-section";

export function SettingsScreen() {
  const t = useTranslations("learning");

  return (
    <div className="anim-fade mx-auto flex w-full max-w-[705px] flex-col gap-7 lg:pt-6">
      <header className="flex flex-col gap-3.5">
        <Link
          href="/learning"
          className="text-foreground/80 flex w-fit items-center gap-1 text-xs"
        >
          <IconArrowLeft size={14} aria-hidden="true" />
          {t("title")}
        </Link>
        <h1 className="text-[26px] leading-tight font-semibold tracking-[-0.02em]">
          {t("settings")}
        </h1>
      </header>

      <AccountSection />
      <AiSection />
      <TriggersSection />
      <ExtensionSection />
      <DeleteAccountSection />
    </div>
  );
}
