import { expect, test, type Page, type Route } from "@playwright/test";

import type {
  LoopDashboard,
  Path,
  PathTeacher,
} from "../src/features/path/schemas";

const createdAt = "2026-11-25T12:00:00Z";
const topic = {
  id: "topic-1",
  position: 0,
  skill: "grammar",
  title: "Past tense",
  goal: "Talk about yesterday",
  unit: "Basics",
  theory: "",
  vocabulary: [],
  status: "completed" as const,
  completedAt: createdAt,
  exercises: { total: 1, done: 1 },
};
const path: Path = {
  id: "program-en",
  language: "en",
  translationLanguage: "ru",
  explainLanguage: "ru",
  level: "A1",
  summary: "English basics",
  status: "active",
  source: "scratch",
  preferences: {},
  weeklyGoalMinutes: 180,
  topics: [topic],
  createdAt,
  archivedAt: null,
};
const teacher: PathTeacher = {
  id: "teacher-en",
  language: "en",
  translationLanguage: "ru",
  explainInTarget: false,
  explainLanguage: "ru",
  createdAt,
  path: { id: path.id, level: path.level, topicsDone: 1, topicsTotal: 1 },
};
const review: LoopDashboard["activities"][number] = {
  id: "review-1",
  type: "review",
  status: "open",
  reason: "due",
  title: "Past tense",
  goal: "Talk about yesterday",
  item: { id: "item-1", key: "past-tense", level: "A1", skill: "grammar" },
  topic: null,
  state: "practiced",
  lapsed: false,
  dueAt: createdAt,
  exercise: {
    id: "exercise-1",
    kind: "fill",
    text: "",
    prompt: "Yesterday I ___ to school.",
    options: [],
  },
  createdAt,
};

test.use({ video: "off" });

async function mockOverview(
  page: Page,
  { withPath, due }: { withPath: boolean; due: boolean },
) {
  let reviewDue = due;
  const attempts: unknown[] = [];

  await page.addInitScript(() => {
    localStorage.setItem("learning_app.token.v1", "test-token");
  });
  await page.route("**/api/v1/**", (route) => {
    const { pathname } = new URL(route.request().url());
    const method = route.request().method();
    const reply = (data: unknown, status = 200) =>
      route.fulfill({
        status,
        json: { code: status < 400 ? 0 : status, message: "success", data },
      });

    if (pathname === "/api/v1/me")
      return reply({ id: "overview-user", first_name: "Learner" });
    if (pathname === "/api/v1/learning/ai/settings")
      return reply({
        provider: "anthropic",
        model: "test",
        level: "medium",
        keyHint: "test",
      });
    if (pathname === "/api/v1/learning/path/teachers")
      return reply([{ ...teacher, path: withPath ? teacher.path : null }]);
    if (pathname === "/api/v1/learning/path" && method === "GET")
      return withPath ? reply(path) : reply(null, 404);
    if (pathname === "/api/v1/learning/path/dashboard")
      return reply({
        language: "en",
        teacherId: teacher.id,
        items: {
          tracked: 1,
          learning: 0,
          practiced: 1,
          mastered: 0,
          lapsed: 0,
        },
        reviews: {
          due: reviewDue ? 1 : 0,
          overdue: 0,
          upcoming: 0,
          nextDueAt: reviewDue ? createdAt : null,
        },
        evidence: { total: 1, last7Days: 0 },
        activities: reviewDue ? [review] : [],
      } satisfies LoopDashboard);
    if (
      pathname === "/api/v1/learning/path/exercises/exercise-1/attempt" &&
      method === "POST"
    ) {
      attempts.push(route.request().postDataJSON());
      reviewDue = false;
      return reply({
        id: "attempt-1",
        exerciseId: "exercise-1",
        answer: "went",
        correct: true,
        mistakes: 0,
        feedback: "Correct.",
        createdAt,
      });
    }
    if (pathname === "/api/v1/learning/path/lessons/active") return reply(null);
    if (pathname === "/api/v1/learning/path/lessons") return reply([]);
    if (pathname === "/api/v1/learning/path/lessons/totals")
      return reply({ today: 0, week: 0, month: 0 });
    return reply(null, 404);
  });

  return attempts;
}

test("a new learner sees empty evidence and can choose a starting point", async ({
  page,
}) => {
  await mockOverview(page, { withPath: false, due: false });
  await page.goto("/learning/overview");

  await expect(
    page.getByRole("heading", { name: "Your learning overview" }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: /Start from scratch/ }),
  ).toBeVisible();
  await expect(page.getByText("Not assessed", { exact: true })).toHaveCount(7);
  await expect(page.getByText("No data yet", { exact: true })).toHaveCount(5);
  await page.getByRole("button", { name: /Start from scratch/ }).click();
  await expect(
    page.getByRole("heading", { name: "Create your learning path" }),
  ).toBeVisible();
});

test("a completed program opens a due review and submits its activity ID", async ({
  page,
}) => {
  const attempts = await mockOverview(page, { withPath: true, due: true });
  await page.goto("/learning/overview");

  await expect(
    page.getByText("Program completed", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Choose extra practice" }),
  ).toHaveAttribute("href", "/learning/path?language=en");
  await page.getByRole("button", { name: "Start review" }).first().click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.getByRole("textbox", { name: "Your answer" }).fill("went");
  await page.getByRole("button", { name: "Check answer" }).click();
  await expect(page.getByRole("status")).toHaveText("Correct");
  expect(attempts).toEqual([{ answer: "went", activityId: "review-1" }]);
  await page.getByRole("button", { name: "Back to overview" }).click();
  await expect(
    page.getByText("You're up to date with your reviews."),
  ).toBeVisible();
});

test("desktop navigation connects the eight sections to stored learning data", async ({ page }) => {
  await mockOverview(page, { withPath: true, due: true });
  await page.goto("/learning/overview");

  const navigation = page.getByRole("navigation", { name: "Learning navigation" });
  await expect(page.getByRole("heading", { name: "Your learning overview" })).toBeVisible();
  await expect(navigation.locator('[aria-current="page"]')).toHaveCount(1);
  await expect(navigation.getByRole("link")).toHaveText([
    "Overview", "Learning Path", "Knowledge Base", "Practice",
    "Assessments", "Progress", "AI Tutor", "Settings",
  ]);

  await navigation.getByRole("link", { name: "Practice" }).click();
  await expect(page.getByRole("heading", { name: "Practice", exact: true })).toBeVisible();
  await expect(page.getByText("Past tense", { exact: true })).toBeVisible();

  await navigation.getByRole("link", { name: "Progress" }).click();
  await expect(page.getByRole("heading", { name: "Progress", exact: true })).toBeVisible();
  await expect(page.getByText("1 saved observations")).toBeVisible();

  await navigation.getByRole("link", { name: "Knowledge Base" }).click();
  await expect(page.getByRole("heading", { name: "Knowledge Base", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Personal notebook" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Open notebook" })).toHaveAttribute("href", "/learning");

  await page.getByRole("link", { name: /Account settings/ }).click();
  await expect(page.getByRole("heading", { name: "Interface and session" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Toggle theme" })).toBeVisible();
  await expect(navigation.locator('[aria-current="page"]')).toHaveText("Settings");
});

test("Telegram width keeps all eight destinations in the mobile menu", async ({ page }) => {
  await mockOverview(page, { withPath: true, due: false });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/learning/overview");

  await page.getByRole("button", { name: "Open navigation" }).click();
  const menu = page.getByRole("dialog");
  await expect(menu.getByRole("navigation", { name: "Learning navigation" })).toBeVisible();
  await expect(menu.locator('[aria-current="page"]')).toHaveCount(1);
  await expect(menu.getByRole("navigation", { name: "Learning navigation" }).getByRole("link")).toHaveCount(8);
  await menu.getByRole("link", { name: "Practice" }).click();
  await expect(page.getByRole("heading", { name: "Practice", exact: true })).toBeVisible();
  await expect(menu).toBeHidden();
});

test("tablet width uses the MagicPath desktop sidebar", async ({ page }) => {
  await mockOverview(page, { withPath: false, due: false });
  await page.setViewportSize({ width: 900, height: 800 });
  await page.goto("/learning/overview");

  const navigation = page.getByRole("navigation", { name: "Learning navigation" });
  await expect(navigation).toBeVisible();
  await expect(navigation.getByRole("link")).toHaveCount(8);
  await expect(page.getByRole("button", { name: "Open navigation" })).toBeHidden();
});

test("switching the language replaces Progress data and the Path destination", async ({ page }) => {
  await mockOverview(page, { withPath: true, due: false });
  const spanish = { ...teacher, id: "teacher-es", language: "es", path: null };
  await page.route("**/api/v1/learning/path/teachers", (route) =>
    route.fulfill({ json: { code: 0, message: "success", data: [teacher, spanish] } }),
  );
  await page.route("**/api/v1/learning/path/dashboard?*", (route) => {
    if (new URL(route.request().url()).searchParams.get("language") !== "es") return route.fallback();
    return route.fulfill({ json: {
      code: 0, message: "success", data: {
        language: "es", teacherId: spanish.id,
        items: { tracked: 0, learning: 0, practiced: 0, mastered: 0, lapsed: 0 },
        reviews: { due: 0, overdue: 0, upcoming: 0, nextDueAt: null },
        evidence: { total: 0, last7Days: 0 }, activities: [],
      } satisfies LoopDashboard,
    } });
  });

  await page.goto("/learning/progress");
  await expect(page.getByText("Learning language: English")).toBeVisible();
  await expect(page.getByText("1 saved observations")).toBeVisible();
  await page.getByRole("button", { name: "Switch language profile" }).click();
  await page.getByRole("menuitem", { name: /Spanish/ }).click();
  await expect(page.getByText("Learning language: Spanish")).toBeVisible();
  await expect(page.getByText("0 saved observations")).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Learning navigation" }).getByRole("link", { name: "Learning Path" }))
    .toHaveAttribute("href", "/learning/path?language=es");
});

test("starting a new plan archives the program and returns to starting choices", async ({ page }) => {
  await mockOverview(page, { withPath: true, due: false });
  let hasPath = true;
  const reply = (route: Route, data: unknown, status = 200) =>
    route.fulfill({ status, json: { code: status < 400 ? 0 : status, message: "success", data } });
  await page.route("**/api/v1/learning/path/teachers", (route) =>
    reply(route, [{ ...teacher, path: hasPath ? teacher.path : null }]),
  );
  await page.route("**/api/v1/learning/path?*", (route) =>
    hasPath ? reply(route, path) : reply(route, null, 404),
  );
  await page.route("**/api/v1/learning/path/reset", (route) => {
    hasPath = false;
    return reply(route, null);
  });
  await page.goto("/learning/path?language=en");
  await page.getByRole("button", { name: "Start a new plan", exact: true }).click();
  await expect(page.getByText(/Previous lessons, answers and results stay available/)).toBeVisible();
  await page.getByRole("button", { name: "Start a new plan; keep history" }).click();
  await expect(page).toHaveURL(/\/learning\/assessment$/);
  await expect(page.getByRole("heading", { name: "Find your starting point" })).toBeVisible();
  await expect(page.getByText("Quick assessment", { exact: true })).toBeVisible();
});

test("Knowledge Base filters published Rule Book items without invented learner status", async ({ page }) => {
  await mockOverview(page, { withPath: true, due: false });
  await page.route("**/api/v1/learning/path/rules?*", (route) =>
    route.fulfill({ json: { code: 0, message: "success", data: [
      { id: "rule-past", level: "A2", title: "Past Simple", unit: "Past Forms" },
      { id: "rule-articles", level: "A1", title: "Articles", unit: "Nouns" },
    ] } }),
  );
  await page.goto("/learning/knowledge?language=en");
  await expect(page.getByText("Published rules: 2")).toBeVisible();
  await expect(page.getByRole("link", { name: /Past Simple/ })).toHaveAttribute("href", "/learning/path/rules/rule-past?language=en");
  await page.getByRole("button", { name: /Past Forms/ }).click();
  await expect(page.getByText("Rules shown: 1")).toBeVisible();
  await expect(page.getByRole("link", { name: /Articles/ })).toHaveCount(0);
  await page.getByRole("button", { name: /All grammar/ }).click();
  await page.getByRole("textbox", { name: "Find a rule" }).fill("articles");
  await expect(page.getByRole("link", { name: /Articles/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Past Simple/ })).toHaveCount(0);
});
