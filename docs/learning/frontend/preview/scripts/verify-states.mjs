import { chromium } from "@playwright/test";

const baseUrl = process.env.DESIGN_PREVIEW_URL ?? "http://127.0.0.1:5187/";
const browser = await chromium.launch({
  headless: true,
  executablePath: process.env.CHROME_PATH || undefined,
  args: ["--no-sandbox"],
});

const states = [
  "concept", "new", "done", "current", "unfinished",
  "due", "insufficient", "missing-ai", "unavailable",
];
const sections = [
  "Overview", "Learning Path", "Knowledge Base", "Practice",
  "Assessments", "Progress", "AI Tutor", "Settings",
];

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

try {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  const errors = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.goto(baseUrl);
  const nav = page.getByRole("navigation", { name: "Learning navigation" });

  for (const state of states) {
    await nav.getByRole("button", { name: "Overview", exact: true }).click();
    await page.locator("#preview-state").selectOption(state);
    for (const section of sections) {
      await nav.getByRole("button", { name: section, exact: true }).click();
      const selector = page.locator(section === "Knowledge Base" ? "#kb-preview-state" : "#preview-state");
      assert(await selector.inputValue() === state, `${section} lost ${state}`);
      if (section === "Overview" && state === "unfinished") {
        await page.getByRole("heading", { name: "Resume your Quick grammar screening" }).waitFor();
        assert(await page.getByRole("heading", { name: "Choose where to begin" }).count() === 0, "Unfinished assessment shows new-learner choices");
      }
      if (section === "Learning Path" && state === "unfinished") {
        await page.getByRole("heading", { name: "Finish or resume your assessment" }).waitFor();
        assert(await page.getByRole("heading", { name: "Your modules" }).count() === 0, "Unfinished assessment shows modules");
      }
      if (section === "Assessments" && state === "unfinished") {
        await page.getByRole("heading", { name: "Assessment paused" }).waitFor();
      }
      if (section === "Practice" && state === "new") {
        await page.getByRole("heading", { name: "Choose how to start practising" }).waitFor();
        assert(await page.getByRole("button", { name: "Start sample activity" }).count() === 0, "Practice starts without a program");
      }
      if (section === "Progress" && (state === "new" || state === "insufficient")) {
        assert(await page.getByText("Sample B1").count() === 0, "Progress shows a sample level in an empty state");
      }
      if (section === "AI Tutor" && state === "missing-ai") {
        await page.getByRole("heading", { name: "Connect an AI provider to use the tutor" }).waitFor();
        assert(await page.getByRole("textbox", { name: "Message AI Tutor" }).count() === 0, "Tutor chat is active without a key");
      }
      if (section === "Settings" && state === "missing-ai") {
        await page.getByText("Not connected", { exact: true }).waitFor();
      }
    }
    console.log(`${state}: 8 sections OK`);
  }

  await nav.getByRole("button", { name: "Overview", exact: true }).click();
  await page.locator("#preview-state").selectOption("current");
  await page.locator("#language-select").selectOption("Spanish");
  await nav.getByRole("button", { name: "Learning Path", exact: true }).click();
  await page.getByRole("heading", { name: "Ser and estar" }).waitFor();
  await nav.getByRole("button", { name: "Progress", exact: true }).click();
  await page.getByRole("heading", { name: "Your Spanish learning progress" }).waitFor();
  await nav.getByRole("button", { name: "Knowledge Base", exact: true }).click();
  await page.getByText("Spanish", { exact: true }).first().waitFor();
  console.log("Spanish language context: OK");

  await page.locator("#language-select").selectOption("");
  for (const section of sections) {
    await nav.getByRole("button", { name: section, exact: true }).click();
    await page.getByRole("heading", { name: "Add your first language" }).waitFor();
  }
  console.log("No-language onboarding gate: 8 sections OK");
  assert(errors.length === 0, errors.join("; "));
} finally {
  await browser.close();
}
