import { chromium } from "@playwright/test";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const output = path.resolve(here, "../../screenshots");
const baseUrl = process.env.DESIGN_PREVIEW_URL ?? "http://127.0.0.1:5187/";
const browser = await chromium.launch({
  headless: true,
  executablePath: process.env.CHROME_PATH || undefined,
  args: ["--no-sandbox"],
});

const desktop = { width: 1440, height: 900 };
const telegram = { width: 390, height: 844 };
const nav = (page) => page.getByRole("navigation", { name: "Learning navigation" });

const shots = [
  { name: "01-overview-active-desktop", viewport: desktop },
  { name: "02-overview-new-desktop", viewport: desktop, state: "new" },
  { name: "03-overview-completed-desktop", viewport: desktop, state: "done" },
  { name: "04-overview-unfinished-assessment-desktop", viewport: desktop, state: "unfinished" },
  { name: "05-overview-due-review-desktop", viewport: desktop, state: "due" },
  { name: "06-overview-insufficient-data-desktop", viewport: desktop, state: "insufficient" },
  { name: "07-overview-missing-ai-key-desktop", viewport: desktop, state: "missing-ai" },
  { name: "08-overview-unavailable-service-desktop", viewport: desktop, state: "unavailable" },
  { name: "09-onboarding-no-language-desktop", viewport: desktop, noLanguage: true },
  { name: "10-learning-path-desktop", viewport: desktop, section: "Learning Path" },
  { name: "11-knowledge-base-desktop", viewport: desktop, section: "Knowledge Base" },
  { name: "12-practice-desktop", viewport: desktop, section: "Practice" },
  { name: "13-assessments-desktop", viewport: desktop, section: "Assessments" },
  { name: "14-progress-desktop", viewport: desktop, section: "Progress" },
  { name: "15-ai-tutor-desktop", viewport: desktop, section: "AI Tutor" },
  { name: "16-settings-desktop", viewport: desktop, section: "Settings" },
  { name: "17-overview-new-telegram", viewport: telegram, state: "new" },
  { name: "18-navigation-telegram", viewport: telegram, menu: true },
  { name: "19-assessments-telegram", viewport: telegram, section: "Assessments" },
  { name: "20-knowledge-base-telegram", viewport: telegram, section: "Knowledge Base" },
];

await mkdir(output, { recursive: true });
try {
  for (const shot of shots) {
    const context = await browser.newContext({ viewport: shot.viewport, deviceScaleFactor: 1 });
    const page = await context.newPage();
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.goto(baseUrl);
    await page.getByRole("heading", { name: "Your learning overview" }).waitFor();

    if (shot.noLanguage) {
      await page.locator("#language-select").selectOption("");
      await page.getByRole("heading", { name: "Add your first language" }).waitFor();
    } else if (shot.state) {
      await page.locator("#preview-state").selectOption(shot.state);
    }
    if (shot.section) {
      if (shot.viewport.width < 768) {
        await page.getByRole("button", { name: "Open menu" }).click();
      }
      await nav(page).getByRole("button", { name: shot.section, exact: true }).click();
    }
    if (shot.menu) {
      await page.getByRole("button", { name: "Open menu" }).click();
      await nav(page).waitFor();
    }
    await page.evaluate(() => document.fonts.ready);
    // Let the navigation hover/selection transition settle before capture.
    await page.waitForTimeout(200);
    if (errors.length) throw new Error(`${shot.name}: ${errors.join("; ")}`);
    await page.screenshot({ path: path.join(output, `${shot.name}.png`) });
    console.log(`${shot.name}.png — ${shot.viewport.width}×${shot.viewport.height}`);
    await context.close();
  }
} finally {
  await browser.close();
}
