import { readFileSync } from "node:fs"
import { randomUUID } from "node:crypto"
import { test as base, expect } from "@playwright/test"

// Seed an account in the isolated database with backend/.learning-fixture.
// Each test creates and removes its own topic and extension tokens.
const sessionFile = process.env.E2E_SESSION_FILE
const session: { token: string } | undefined = sessionFile
  ? JSON.parse(readFileSync(sessionFile, "utf8"))
  : undefined
const backend = process.env.E2E_BACKEND_URL ?? "http://localhost:8181"

const test = base.extend<{ topic: { id: string; name: string } }>({
  topic: async ({ page, request, context, baseURL }, runTest) => {
    test.skip(
      !session,
      "Set E2E_SESSION_FILE to a seeded test account JSON file",
    )
    const headers = { Authorization: `Bearer ${session!.token}` }
    const response = await request.post(`${backend}/api/v1/learning/topics`, {
      headers,
      data: {
        name: `Learning QA ${randomUUID()}`,
        language: "en",
        translationLanguage: "ru",
      },
    })
    expect(response.status()).toBe(201)
    const topic = (await response.json()).data as { id: string; name: string }
    await context.addCookies([
      { name: "NEXT_LOCALE", value: "en", url: baseURL! },
    ])
    await page.addInitScript((token) => {
      localStorage.setItem("learning_app.token.v1", token)
    }, session!.token)
    try {
      await runTest(topic)
    } finally {
      await request.delete(`${backend}/api/v1/learning/topics/${topic.id}`, {
        headers,
      })
    }
  },
})

test.use({ video: "off" })

test("editing a filtered entry keeps its details open", async ({
  page,
  topic,
}) => {
  await page.goto(`/learning/${topic.id}`)
  await expect(page.getByRole("heading", { name: topic.name })).toBeVisible()
  await page
    .getByRole("button", { name: "Add entry", exact: true })
    .first()
    .click()
  let dialog = page.getByRole("dialog")
  await dialog
    .getByLabel("Word or phrase", { exact: true })
    .fill("hit the road")
  await dialog
    .getByLabel("Translation", { exact: true })
    .fill("отправиться в путь")
  await dialog
    .getByLabel("How to use it", { exact: true })
    .fill("We should hit the road.")
  await dialog.getByRole("button", { name: "Save", exact: true }).click()
  await expect(dialog).toBeHidden()

  const filters = page
    .getByRole("group", { name: "Learning status", exact: true })
    .first()
  await filters.getByRole("button", { name: "Inbox", exact: true }).click()
  await page.getByRole("button", { name: /hit the road/ }).click()
  const details = page.locator("aside").filter({
    has: page.getByRole("heading", { name: "hit the road", exact: true }),
  })
  await expect(
    details.getByText("We should hit the road.", { exact: true }),
  ).toBeVisible()
  await details.getByRole("button", { name: "Known", exact: true }).click()
  await expect(page.getByText("No matches", { exact: true })).toBeVisible()
  await expect(
    details.getByRole("button", { name: "Known", exact: true }),
  ).toHaveAttribute("aria-pressed", "true")
  await details.getByRole("button", { name: "Edit entry", exact: true }).click()
  dialog = page.getByRole("dialog")
  await dialog
    .getByLabel("Word or phrase", { exact: true })
    .fill("hit the road again")
  await dialog.getByRole("button", { name: "Save", exact: true }).click()
  await expect(dialog).toBeHidden()
  await expect(
    page
      .locator("aside")
      .getByRole("heading", { name: "hit the road again", exact: true }),
  ).toBeVisible()

  await page
    .locator("aside")
    .getByRole("button", { name: "Delete entry", exact: true })
    .click()
  await page
    .locator("aside")
    .getByRole("button", { name: "Delete", exact: true })
    .click()
  await expect(
    page.getByText("Select a word or phrase to open its details."),
  ).toBeVisible()
  await page.getByRole("button", { name: "Delete topic", exact: true }).click()
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Delete", exact: true })
    .click()
  await expect(page).toHaveURL(/\/learning$/)
  await expect(
    page.getByRole("link", { name: new RegExp(topic.name) }),
  ).toHaveCount(0)
})

test("mobile details stay open after changing a filtered status and editing", async ({
  page,
  topic,
}) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(`/learning/${topic.id}`)
  await page
    .getByRole("button", { name: "Add entry", exact: true })
    .last()
    .click()
  let dialog = page.getByRole("dialog")
  await dialog.getByLabel("Word or phrase", { exact: true }).fill("figure out")
  await dialog.getByRole("button", { name: "Save", exact: true }).click()
  await expect(dialog).toBeHidden()
  await page
    .getByRole("group", { name: "Learning status", exact: true })
    .getByRole("button", { name: "Inbox", exact: true })
    .click()
  await page.getByRole("button", { name: /figure out/ }).click()
  const status = page.getByRole("group", {
    name: "Learning status",
    exact: true,
  })
  await status.getByRole("button", { name: "Learning", exact: true }).click()
  await expect(
    status.getByRole("button", { name: "Learning", exact: true }),
  ).toHaveAttribute("aria-pressed", "true")
  await page.getByRole("button", { name: "Edit entry", exact: true }).click()
  dialog = page.getByRole("dialog")
  await dialog.getByLabel("Translation", { exact: true }).fill("разобраться")
  await dialog.getByRole("button", { name: "Save", exact: true }).click()
  await expect(dialog).toBeHidden()
  await expect(page.getByText("разобраться", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: topic.name, exact: true }).click()
  await expect(page.getByText("No matches", { exact: true })).toBeVisible()
})

test("mobile entry screen marks an entry known and deletes it", async ({
  page,
  topic,
}) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto(`/learning/${topic.id}`)
  await page
    .getByRole("button", { name: "Add entry", exact: true })
    .last()
    .click()
  const dialog = page.getByRole("dialog")
  await dialog.getByLabel("Word or phrase", { exact: true }).fill("catch up")
  await dialog.getByRole("button", { name: "Save", exact: true }).click()
  await expect(dialog).toBeHidden()
  await page.getByRole("button", { name: /catch up/ }).click()
  await expect(page.getByRole("navigation")).toHaveCount(0)
  await page
    .getByRole("button", { name: "Mark as known", exact: true })
    .click()
  await expect(
    page
      .getByRole("group", { name: "Learning status", exact: true })
      .getByRole("button", { name: "Known", exact: true }),
  ).toHaveAttribute("aria-pressed", "true")
  await expect(
    page.getByRole("button", { name: "Move back to Learning", exact: true }),
  ).toBeVisible()
  await page.getByRole("button", { name: "Delete entry", exact: true }).click()
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Delete", exact: true })
    .click()
  await expect(page.getByRole("button", { name: /catch up/ })).toHaveCount(0)
  await expect(page.getByRole("heading", { name: topic.name })).toBeVisible()
})

test("extension token delivers a capture and revocation blocks further requests", async ({
  page,
  context,
  request,
  topic,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"])
  await page.goto("/learning/tokens")
  await page.getByRole("button", { name: "Create token", exact: true }).click()
  let dialog = page.getByRole("dialog")
  const name = `Chrome QA ${randomUUID()}`
  await dialog.getByLabel("Name", { exact: true }).fill(name)
  await dialog.getByRole("radio", { name: topic.name, exact: true }).click()
  await dialog
    .getByRole("button", { name: "Create token", exact: true })
    .click()
  await expect(
    page.getByRole("dialog", { name: "Token created", exact: true }),
  ).toBeVisible()
  await expect(dialog.getByText(/^bml_[\w-]{8}•+$/)).toBeVisible()
  await dialog.getByRole("button", { name: "Show token", exact: true }).click()
  const token = (await dialog.getByText(/^bml_[\w-]{43}$/).textContent()) ?? ""
  expect(token).toMatch(/^bml_[\w-]{43}$/)
  await dialog.getByRole("button", { name: "Done", exact: true }).click()
  await expect(dialog).toBeHidden()
  const headers = {
    Authorization: `Bearer ${token}`,
    "Idempotency-Key": randomUUID(),
  }
  try {
    const response = await request.post(`${backend}/api/v1/learning/captures`, {
      headers,
      data: { topicId: topic.id, text: "serendipity", kind: "word" },
    })
    expect(response.status()).toBe(202)
    const receipt = (await response.json()).data as { id: string }
    await expect
      .poll(async () => {
        const status = await request.get(
          `${backend}/api/v1/learning/captures/${receipt.id}`,
          { headers },
        )
        return (await status.json()).data.status
      })
      .toBe("done")
    const denied = await request.get(
      `${backend}/api/v1/learning/entries?topicId=${topic.id}`,
      { headers },
    )
    expect(denied.status()).toBe(403)
    await page.goto(`/learning/${topic.id}`)
    await expect(
      page.getByRole("button", { name: /serendipity/ }),
    ).toBeVisible()
    await page.goto("/learning/tokens")
    const card = page
      .locator("div")
      .filter({ has: page.getByRole("heading", { name, exact: true }) })
      .filter({
        has: page.getByRole("button", { name: "Copy token", exact: true }),
      })
      .last()
    await expect(card.getByText(/^bml_[\w-]{8}•+$/)).toBeVisible()
    await expect(card.getByText(token, { exact: true })).toHaveCount(0)
    await card.getByRole("button", { name: "Show token", exact: true }).click()
    await expect(card.getByText(token, { exact: true })).toBeVisible()
    await card.getByRole("button", { name: "Hide token", exact: true }).click()
    await expect(card.getByText(token, { exact: true })).toHaveCount(0)
    await card.getByRole("button", { name: "Copy token", exact: true }).click()
    await expect(card.getByRole("status")).toHaveText("Copied")
    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
      token,
    )
    await card
      .getByRole("button", { name: "Revoke token", exact: true })
      .click()
    dialog = page.getByRole("dialog")
    await dialog
      .getByRole("button", { name: "Revoke token", exact: true })
      .click()
    await expect(dialog).toBeHidden()
    await expect(page.getByRole("heading", { name, exact: true })).toHaveCount(
      0,
    )
    const revoked = await request.get(`${backend}/api/v1/learning/topics`, {
      headers,
    })
    expect(revoked.status()).toBe(401)
  } finally {
    const sessionHeaders = { Authorization: `Bearer ${session!.token}` }
    const list = await request.get(`${backend}/api/v1/learning/tokens`, {
      headers: sessionHeaders,
    })
    for (const item of (await list.json()).data as {
      id: string
      name: string
    }[]) {
      if (item.name === name) {
        await request.delete(`${backend}/api/v1/learning/tokens/${item.id}`, {
          headers: sessionHeaders,
        })
      }
    }
  }
})
