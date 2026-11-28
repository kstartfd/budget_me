import { readFileSync } from "node:fs"
import { expect, test } from "@playwright/test"

const sessionFile = process.env.E2E_SESSION_FILE
const session: { token: string } | undefined = sessionFile
  ? JSON.parse(readFileSync(sessionFile, "utf8"))
  : undefined

test.use({ video: "off" })

for (const viewport of [
  { name: "desktop", width: 1440, height: 900 },
  { name: "telegram", width: 390, height: 844 },
]) test(`${viewport.name}: an answer survives a reload and a partial result builds the learning path`, async ({
  page,
  request,
  context,
  baseURL,
}) => {
  test.skip(!session, "Set E2E_SESSION_FILE to a seeded test account JSON file")
  await page.setViewportSize(viewport)
  const token = session!.token
  const headers = { Authorization: `Bearer ${token}` }
  const path = "/api/v1/learning/path"
  const profiles = await request.get(`${path}/teachers`, { headers })
  expect((await profiles.json()).data).toEqual([])
  let teacher: { id: string } | undefined

  try {
    await context.addCookies([
      { name: "NEXT_LOCALE", value: "en", url: baseURL! },
    ])
    await page.addInitScript((value) => {
      localStorage.setItem("learning_app.token.v1", value)
    }, token)
    if (viewport.name === "desktop") {
      await page.goto("/learning/start")
      await page.getByRole("radio", { name: "Russian", exact: true }).click()
      await page.getByRole("button", { name: "Continue", exact: true }).click()
      await page.getByRole("radio", { name: "English", exact: true }).click()
      for (let step = 0; step < 3; step++)
        await page.getByRole("button", { name: "Continue", exact: true }).click()
      const created = page.waitForResponse((response) => response.request().method() === "POST" && /\/teachers$/.test(new URL(response.url()).pathname))
      await page.getByRole("button", { name: "Create profile and start assessment" }).click()
      const response = await created
      expect(response.status()).toBe(201)
      teacher = (await response.json()).data as { id: string }
      await expect(page).toHaveURL(/\/learning\/assessment$/)
    } else {
      const response = await request.post(`${path}/teachers`, {
        headers,
        data: { language: "en", translationLanguage: "ru" },
      })
      expect(response.status()).toBe(201)
      teacher = (await response.json()).data as { id: string }
      await page.goto("/learning/assessment")
    }
    await expect(
      page.getByRole("heading", { name: "Find your starting point" }),
    ).toBeVisible()
    await expect(
      page.getByRole("radio", { name: /Full assessment/ }),
    ).toBeDisabled()
    await expect(page.getByRole("radio", { name: /Individual skill assessment/ })).toBeDisabled()
    await page
      .getByText("Select an approximate starting point", { exact: true })
      .click()
    await page
      .getByRole("combobox", { name: "Approximate level" })
      .selectOption("B1")
    await page.getByRole("button", { name: "Save starting point" }).click()
    await expect(page.getByText("B1 · Self-reported")).toBeVisible()
    await page.getByRole("button", { name: "Back", exact: true }).click()
    await page.getByText("Quick assessment", { exact: true }).click()
    await page.getByRole("button", { name: "Continue", exact: true }).click()
    await page.getByRole("button", { name: "Start assessment" }).click()
    await expect(page.getByText("Question 1 · 0 answered")).toBeVisible()
    await expect(page.getByRole("link", { name: "Assessments", exact: true })).toBeVisible()
    const form = page.getByRole("form")
    await expect(form.getByRole("heading")).toBeFocused()
    await page.keyboard.press("Tab")
    await expect(form.locator(":focus")).toHaveCount(1)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    if (process.env.E2E_SCREENSHOT_DIR) {
      await page.waitForTimeout(300) // capture after the page transition
      await page.screenshot({ path: `${process.env.E2E_SCREENSHOT_DIR}/p316-${viewport.name}-question.png` })
    }
    const answer = form.getByRole("textbox", { name: "Your answer" })
    if (await answer.count()) await answer.fill("test answer")
    else await form.getByRole("radio").first().check()
    const answerResponse = page.waitForResponse((response) =>
      response.request().method() === "POST" && /\/tests\/[^/]+\/answers$/.test(new URL(response.url()).pathname),
    )
    await form.getByRole("button", { name: "Continue" }).click()
    const submitted = await answerResponse
    const replay = await request.post(new URL(submitted.url()).pathname, {
      headers,
      data: submitted.request().postDataJSON(),
    })
    expect(replay.status()).toBe(200)
    const replayData = (await replay.json()).data
    expect(replayData.assessment.answered).toBe(1)
    expect(replayData.assessment.sections).toEqual([
      expect.objectContaining({ skill: "grammar", answered: 1, status: "in_progress" }),
    ])
    await expect(page.getByText("Question 2 · 1 answered")).toBeVisible()
    const nextQuestion = await form.getByRole("heading").innerText()
    if (viewport.name === "desktop") {
      await page.getByRole("link", { name: "Overview", exact: true }).click()
      await expect(page).toHaveURL(/\/learning\/overview$/)
      await page.getByRole("link", { name: "Assessments", exact: true }).click()
      await expect(page).toHaveURL(/\/learning\/assessment$/)
      await page.getByRole("button", { name: "Resume assessment" }).click()
      await expect(form.getByRole("heading")).toHaveText(nextQuestion)
    }

    await page.getByRole("button", { name: "Save and exit" }).click()
    await expect(
      page.getByText("Assessment paused", { exact: true }),
    ).toBeVisible()
    await page.reload()
    await page.getByRole("button", { name: "Restart assessment…" }).click()
    await page
      .getByRole("dialog")
      .getByRole("button", { name: "Keep my answers" })
      .click()
    await page.getByRole("button", { name: "Resume assessment" }).click()
    await expect(page.getByText("Question 2 · 1 answered")).toBeVisible()
    await expect(form.getByRole("heading")).toHaveText(nextQuestion)

    await form.getByRole("button", { name: "I don’t know" }).click()
    await expect(page.getByText("Question 3 · 2 answered")).toBeVisible()
    await page.getByRole("button", { name: "Save and exit" }).click()
    await page
      .getByRole("button", { name: "Finish with saved answers" })
      .click()
    await expect(
      page.getByRole("heading", { name: "Assessment results" }),
    ).toBeVisible()
    await expect(page.getByText("English · Partial results")).toBeVisible()
    await expect(
      page.getByText("Not assessed", { exact: true }).first(),
    ).toBeVisible()
    await expect(page.getByText(/preliminary grammar evidence/)).toBeVisible()
    await expect(page.getByRole("button", { name: "Complete another section" })).toHaveCount(0)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    if (process.env.E2E_SCREENSHOT_DIR) {
      await page.waitForTimeout(300)
      await page.screenshot({ path: `${process.env.E2E_SCREENSHOT_DIR}/p316-${viewport.name}-result.png` })
    }
    await page
      .getByRole("button", { name: "Continue to learning path" })
      .click()
    await page.getByRole("button", { name: "Create my learning path" }).click()
    await expect(page).toHaveURL(/\/learning\/overview$/)
    await expect
      .poll(async () => {
        const current = await request.get(`${path}?language=en`, { headers })
        return (await current.json()).data?.source
      })
      .toBe("assessment")
  } finally {
    if (teacher) await request.delete(`${path}/teachers/${teacher.id}`, { headers })
  }
})

test("capabilities and UI offer grammar Quick only for English and Spanish", async ({ request, page, context, baseURL }) => {
  test.skip(!session, "Set E2E_SESSION_FILE to a seeded test account JSON file")
  const endpoint = "/api/v1/learning/path/assessments/capabilities"
  const headers = { Authorization: `Bearer ${session!.token}` }
  for (const language of ["en", "es"]) {
    const response = await request.get(`${endpoint}?language=${language}`, { headers })
    expect(response.status()).toBe(200)
    expect((await response.json()).data.available).toEqual([
      expect.objectContaining({ type: "quick_screening", skills: ["grammar"] }),
    ])
  }
  const unavailable = await request.get(`${endpoint}?language=de`, { headers })
  expect((await unavailable.json()).data.available).toEqual([])
  expect((await request.get(`${endpoint}?language=en`)).status()).toBe(401)

  await context.addCookies([{ name: "NEXT_LOCALE", value: "en", url: baseURL! }])
  await page.addInitScript((token) => localStorage.setItem("learning_app.token.v1", token), session!.token)
  const teachers = "/api/v1/learning/path/teachers"
  for (const language of ["es", "de"]) {
    const created = await request.post(teachers, { headers, data: { language, translationLanguage: "en" } })
    expect(created.status()).toBe(201)
    const teacher = (await created.json()).data as { id: string }
    try {
      await page.goto("/learning/assessment")
      const quick = page.getByRole("radio", { name: /Quick assessment/ })
      if (language === "es") {
        await expect(quick).toBeEnabled()
        await page.getByRole("button", { name: "Continue", exact: true }).click()
        const start = page.waitForResponse((response) => response.request().method() === "POST" && /\/tests$/.test(new URL(response.url()).pathname))
        await page.getByRole("button", { name: "Start assessment" }).click()
        const payload = (await (await start).json()).data
        expect(payload.language).toBe("es")
        expect(payload.assessment.sections[0]).toEqual(expect.objectContaining({ skill: "grammar", status: "in_progress" }))
        await expect(page.getByText("Question 1 · 0 answered")).toBeVisible()
      } else {
        await expect(quick).toBeDisabled()
        await expect(page.getByText("Quick assessment is currently available for English and Spanish.")).toBeVisible()
      }
    } finally {
      await request.delete(`${teachers}/${teacher.id}`, { headers })
    }
  }
})
