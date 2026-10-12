import type { Metadata, Viewport } from "next"
import { GeistMono } from "geist/font/mono"
import { GeistSans } from "geist/font/sans"
import { NextIntlClientProvider } from "next-intl"
import { getLocale, getMessages } from "next-intl/server"

import "./globals.css"
import { AppProviders } from "@/app/providers"
import { AppShell } from "@/components/app-shell"
import { ThemeProvider } from "@/components/theme-provider"
import { cn } from "@/lib/utils"

export const metadata: Metadata = {
  title: "budget_me",
  description: "Personal budgeting, right inside Telegram.",
}

// Telegram webview: lock the scale and extend under the home indicator so the
// bottom nav can pad itself with safe-area insets.
export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
  viewportFit: "cover",
}

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  const locale = await getLocale()
  const messages = await getMessages()

  return (
    <html
      lang={locale}
      suppressHydrationWarning
      className={cn("antialiased", GeistSans.variable, GeistMono.variable)}
    >
      <head>
        {/* Telegram Mini App runtime — a plain synchronous script in <head> so
            window.Telegram.WebApp exists before React hydrates. Served from our
            origin (vendored copy in public/): telegram.org is not reachable from
            every network the webview runs on, and a hanging blocking remote
            script means a blank app. */}
        {/* eslint-disable-next-line @next/next/no-sync-scripts -- intentional: the
            Telegram runtime must exist before hydration, and it's a small local
            vendored file, so there's no network stall to worry about. */}
        <script src="/telegram-web-app.js" />
      </head>
      <body>
        <NextIntlClientProvider messages={messages}>
          {/* Design default is dark; the header toggle switches explicitly. */}
          <ThemeProvider defaultTheme="dark" enableSystem={false}>
            <AppProviders>
              <AppShell>{children}</AppShell>
            </AppProviders>
          </ThemeProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  )
}
