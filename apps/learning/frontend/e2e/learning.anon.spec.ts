import { expect, test } from "@playwright/test"

const topicId = "00000000-0000-4000-8000-000000000001"

test.use({ video: "off" })

test("chat renders Markdown emphasis without asterisks", async ({
  page,
  baseURL,
}) => {
  await page.addInitScript(() => {
    localStorage.setItem("learning_app.token.v1", "test-token")
  })
  await page.route("**/api/v1/**", (route) => {
    const path = new URL(route.request().url()).pathname
    const reply = (data: unknown, status = 200) =>
      route.fulfill({
        status,
        json: { code: status === 200 ? 0 : 404, message: "success", data },
      })

    if (path === "/api/v1/me") {
      return reply({ id: "user-1", first_name: "Learner" })
    }
    if (path === "/api/v1/learning/topics") {
      return reply([
        {
          id: topicId,
          name: "English",
          language: "en",
          translationLanguage: "ru",
        },
      ])
    }
    if (path === `/api/v1/learning/topics/${topicId}/chat`) {
      return reply({
        available: true,
        teacher: true,
        messages: [
          {
            id: 1,
            role: "assistant",
            content: "This is **important** and *useful*.",
            createdAt: new Date().toISOString(),
          },
        ],
        budget: null,
        daily: null,
      })
    }
    if (path === "/api/v1/learning/path") return reply(null, 404)
    if (path === "/api/v1/learning/path/history") return reply([])
    return reply(null, 404)
  })

  await page.goto("/learning/chat")
  const answer = page.locator("p").filter({
    has: page.locator("em", { hasText: "important" }),
  })
  await expect(answer).toHaveText("This is important and useful.")
  await expect(answer.locator("em")).toHaveText("important")
  await expect(answer.locator("strong")).toHaveText("useful")

  const sidebar = page.locator("aside").first()
  await expect(
    sidebar.getByRole("link", { name: "Topics", exact: true }),
  ).toHaveCount(2)
  await page.context().addCookies([
    { name: "NEXT_LOCALE", value: "ru", url: baseURL! },
  ])
  await page.reload()
  await expect(
    sidebar.getByRole("link", { name: "Топики", exact: true }),
  ).toHaveCount(2)
})
