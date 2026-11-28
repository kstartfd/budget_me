import { expect, test, type Page } from "@playwright/test"

import type { Path, PathTeacher, PathTest } from "../src/features/path/schemas"

const createdAt = "2026-11-25T12:00:00Z"
const spanish: PathTeacher = {
  id: "teacher-es",
  language: "es",
  translationLanguage: "en",
  explainInTarget: false,
  explainLanguage: "en",
  createdAt,
  path: null,
}
const prepared: PathTest = {
  id: "prepared-test",
  kind: "placement",
  status: "done",
  language: "es",
  translationLanguage: "en",
  level: "A2",
  levelReached: "A2",
  passed: false,
  questions: [],
  asked: 0,
  correct: 0,
  needed: 0,
  createdAt,
  completedAt: createdAt,
}
const program: Path = {
  id: "program-es",
  language: "es",
  translationLanguage: "en",
  explainLanguage: "en",
  level: "A2",
  summary: "Spanish basics",
  status: "active",
  source: "self_reported",
  preferences: {},
  weeklyGoalMinutes: 180,
  topics: [
    {
      id: "current-topic",
      position: 0,
      skill: "grammar",
      title: "Forming questions",
      goal: "Ask simple questions",
      unit: "Basics",
      theory: "",
      vocabulary: [],
      status: "not_started",
      completedAt: null,
      exercises: { total: 3, done: 0 },
    },
  ],
  createdAt,
  archivedAt: null,
}

test.use({ video: "off" })

async function mockSetup(
  page: Page,
  {
    aiReady = false,
    teachers = [],
  }: { aiReady?: boolean; teachers?: PathTeacher[] } = {},
) {
  const calls = { create: 0, prepare: 0, assess: 0, build: 0 }
  let built = false

  await page.addInitScript(() => {
    localStorage.setItem("learning_app.token.v1", "test-token")
  })
  await page.route("**/api/v1/**", (route) => {
    const { pathname } = new URL(route.request().url())
    const method = route.request().method()
    const reply = (data: unknown, status = 200, message = "success") =>
      route.fulfill({
        status,
        json: { code: status < 400 ? 0 : status, message, data },
      })

    if (
      method === "PATCH" &&
      pathname.startsWith("/api/v1/learning/path/teachers/")
    ) {
      const teacher = teachers.find(({ id }) => pathname.endsWith(`/${id}`))
      if (!teacher) return reply(null, 404)
      const { explainInTarget } = route.request().postDataJSON()
      const updated = {
        ...teacher,
        explainInTarget,
        explainLanguage: explainInTarget
          ? teacher.language
          : teacher.translationLanguage,
      }
      teachers = teachers.map((item) =>
        item.id === updated.id ? updated : item,
      )
      return reply(updated)
    }

    switch (pathname) {
      case "/api/v1/me":
        return reply({ id: "user-1", first_name: "Learner" })
      case "/api/v1/learning/ai/settings":
        return reply(
          aiReady
            ? {
                provider: "anthropic",
                model: "claude-sonnet-5-5",
                level: "medium",
                keyHint: "test",
              }
            : null,
        )
      case "/api/v1/learning/path/teachers": {
        if (method === "GET") return reply(teachers)
        calls.create++
        const { language, translationLanguage } = route.request().postDataJSON()
        const teacher = {
          ...spanish,
          id: `teacher-${language}`,
          language,
          translationLanguage,
          explainLanguage: translationLanguage,
        }
        teachers = [...teachers, teacher]
        return reply(teacher, 201)
      }
      case "/api/v1/learning/path/levels":
        calls.prepare++
        return reply(prepared, 201)
      case "/api/v1/learning/path/tests":
        calls.assess++
        return reply(null, 503)
      case "/api/v1/learning/path/tests/prepared-test/analyze":
        calls.build++
        if (calls.build === 1)
          return reply(null, 503, "Planner is temporarily unavailable")
        return reply(
          {
            id: "build-job",
            testId: prepared.id,
            status: "running",
            error: "",
            path: null,
          },
          202,
        )
      case "/api/v1/learning/path/jobs/build-job":
        return reply({
          id: "build-job",
          testId: prepared.id,
          status: built ? "done" : "running",
          error: "",
          path: built ? program : null,
        })
      case "/api/v1/learning/path":
        return built ? reply(program) : reply(null, 404)
      case "/api/v1/learning/path/lessons":
      case "/api/v1/learning/path/history":
        return reply([])
      case "/api/v1/learning/path/lessons/active":
        return reply(null)
      default:
        return reply(null, 404, "Not found")
    }
  })

  return {
    calls,
    completeBuild() {
      built = true
      teachers = [
        {
          ...spanish,
          path: {
            id: program.id,
            level: program.level,
            topicsDone: 0,
            topicsTotal: program.topics.length,
          },
        },
      ]
    },
  }
}

async function advanceWizard(page: Page) {
  await page.goto("/learning/start")
  for (let step = 0; step < 4; step++) {
    await page.getByRole("button", { name: "Continue", exact: true }).click()
  }
  await expect(
    page.getByRole("heading", { name: "Where do you want to start?" }),
  ).toBeVisible()
}

test("the wizard picks a native language, a study language and an explanation mode", async ({
  page,
}) => {
  await mockSetup(page)
  await page.goto("/learning/start")

  await expect(
    page.getByRole("heading", { name: "Choose your native language" }),
  ).toBeVisible()
  await page.getByRole("radio", { name: "Russian", exact: true }).click()
  const next = page.getByRole("button", { name: "Continue", exact: true })
  await next.click()
  await expect(
    page.getByRole("heading", { name: "Which language do you want to learn?" }),
  ).toBeVisible()
  await expect(
    page.getByRole("radio", { name: "Russian", exact: true }),
  ).toHaveCount(0)
  await page.getByRole("radio", { name: "English", exact: true }).click()

  const explainInEnglish = page.getByRole("checkbox", {
    name: "Explain in English",
  })
  await expect(explainInEnglish).not.toBeChecked()
  await expect(explainInEnglish).toHaveAccessibleDescription(
    "The default language for explanations is Russian.",
  )
  await explainInEnglish.click()
  await page.getByRole("button", { name: "Back", exact: true }).click()
  await expect(
    page.getByRole("radio", { name: "Russian", exact: true }),
  ).toHaveAttribute("aria-checked", "true")
  await next.click()
  await expect(explainInEnglish).toBeChecked()
  await expect(page.getByRole("combobox")).toHaveCount(0)
  for (let step = 0; step < 3; step++) await next.click()

  const creation = page.waitForRequest(
    (request) =>
      request.method() === "POST" &&
      request.url().endsWith("/learning/path/teachers"),
  )
  const update = page.waitForRequest(
    (request) =>
      request.method() === "PATCH" &&
      request.url().endsWith("/learning/path/teachers/teacher-en"),
  )
  await page.getByRole("button", { name: "Create profile and start assessment" }).click()
  expect((await creation).postDataJSON()).toEqual({
    language: "en",
    translationLanguage: "ru",
  })
  expect((await update).postDataJSON()).toEqual({ explainInTarget: true })
  await expect(
    page.getByRole("heading", { name: "Find your starting point" }),
  ).toBeVisible()
  await expect(page).toHaveURL(/\/learning\/assessment$/)
})

test("English is also available from the teacher list", async ({ page }) => {
  await mockSetup(page)
  await page.goto("/learning/path")

  const target = page.getByRole("combobox", { name: "Learning language" })
  await target.click()
  await page.getByRole("option", { name: "English", exact: true }).click()
  const add = page.getByRole("button", { name: "Add", exact: true })
  await expect(add).toBeDisabled()

  await page.getByRole("combobox", { name: "Your language" }).click()
  await page.getByRole("option", { name: "Russian", exact: true }).click()
  await expect(target).toHaveText("English")
  await add.click()
  await expect(
    page.getByRole("heading", { name: "Teacher: English" }),
  ).toBeVisible()
})

test("a first profile and its starting point are saved without an AI provider", async ({
  page,
}) => {
  const { calls } = await mockSetup(page)
  await page.goto("/")
  await expect(
    page.getByRole("heading", { name: "Add your first language" }),
  ).toBeVisible()
  await advanceWizard(page)
  await page.getByRole("radio", { name: /Start from scratch/ }).click()
  await page.getByRole("button", { name: "Save profile", exact: true }).click()

  await expect(page).toHaveURL(/\/learning\/overview$/)
  await expect(
    page.getByRole("heading", { name: "Create your learning path" }),
  ).toBeVisible()
  await expect(
    page.getByText("Start your program with the fundamentals."),
  ).toBeVisible()
  await expect(
    page.getByRole("button", { name: "Create program", exact: true }),
  ).toBeDisabled()
  await page.reload()
  await expect(
    page.getByText("Start your program with the fundamentals."),
  ).toBeVisible()
  expect(calls).toEqual({ create: 1, prepare: 0, assess: 0, build: 0 })
})

test("retrying the wizard after a build request fails reuses the prepared test", async ({
  page,
}) => {
  const { calls, completeBuild } = await mockSetup(page, { aiReady: true })
  await advanceWizard(page)
  await page.getByRole("radio", { name: /Choose an approximate level/ }).click()
  await page
    .getByRole("button", { name: "Create profile and learning path" })
    .click()

  await expect(
    page.getByText("Planner is temporarily unavailable"),
  ).toBeVisible()
  await expect(
    page.getByRole("heading", { name: "Where do you want to start?" }),
  ).toBeVisible()
  await expect(
    page.getByRole("radio", { name: /Start from scratch/ }),
  ).toBeDisabled()
  await page.getByRole("button", { name: "Try again", exact: true }).click()

  await expect(page).toHaveURL(/\/learning\/overview$/)
  await expect(page.getByText("Creating your Spanish program")).toBeVisible()
  expect(calls).toEqual({ create: 1, prepare: 1, assess: 0, build: 2 })

  completeBuild()
  await page
    .getByRole("link", { name: "Continue learning", exact: true })
    .click()
  await expect(page).toHaveURL(
    /\/learning\/path\?language=es&topic=current-topic$/,
  )
  await expect(
    page.getByRole("button", { name: /Forming questions/ }),
  ).toHaveAttribute("aria-expanded", "true")
})

test("switching the language in the sidebar opens its teacher", async ({
  page,
}) => {
  await mockSetup(page, {
    teachers: [spanish, { ...spanish, id: "teacher-fr", language: "fr" }],
  })
  await page.goto("/learning/path?language=es")
  await expect(
    page.getByRole("heading", { name: "Teacher: Spanish" }),
  ).toBeVisible()
  await page.getByRole("button", { name: "Switch language profile" }).click()
  await page.getByRole("menuitem", { name: /French/ }).click()

  await expect(page).toHaveURL(/\/learning\/path\?language=fr$/)
  await expect(
    page.getByRole("heading", { name: "Teacher: French" }),
  ).toBeVisible()
  await page.getByRole("button", { name: "All teachers" }).click()
  await expect(page).toHaveURL(/\/learning\/path$/)
})
