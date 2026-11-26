# Local Learning design preview

This Vite app imports the canonical TSX files in `../MagicPath/` directly. It does not call the Learning backend or MagicPath. All displayed learner progress, questions, answers, levels, and insights are **sample content**.

From this directory, run `npm ci && npm run dev`, then open `http://127.0.0.1:5187/`. Build with `npm run build`. In another terminal, regenerate the checked-in screenshots with `CHROME_PATH=/usr/bin/google-chrome npm run capture`. Use `DESIGN_PREVIEW_URL=http://127.0.0.1:<port>/` only if you deliberately run Vite on another port.

To verify an isolated copy as a clean checkout, copy `docs/learning/frontend/` to a temporary directory without any `node_modules` or `dist`, run `npm ci` inside its `preview/` subdirectory, then run `npm run build` and `npm run dev`. The source imports resolve relative to this directory, so no application checkout, credentials, backend, or hosted design service is needed. The capture script requires an installed Chromium/Chrome binary or Playwright's bundled Chromium.
