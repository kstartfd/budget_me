# Production image for the Next.js 16 frontend (standalone output).
# Build context is the frontend/ directory (see infra/compose/prod.yml:
# context ../../frontend) — the repo-root .dockerignore excludes frontend/,
# so a repo-root context would ship no source. All COPY paths are therefore
# relative to frontend/. next.config.mjs must set output: "standalone".

# ── deps ─────────────────────────────────────────────────────────────────────
FROM node:22-alpine AS deps
WORKDIR /app

RUN corepack enable
COPY package.json pnpm-lock.yaml ./
# --ignore-scripts: skip dependency build scripts (sharp/@swc/core/…) — pnpm 10
# blocks them by default and next build doesn't need them (Next bundles swc).
RUN pnpm install --frozen-lockfile --ignore-scripts

# ── builder ──────────────────────────────────────────────────────────────────
FROM node:22-alpine AS builder
WORKDIR /app

RUN corepack enable
# Cap the heap so `next build` does not OOM on a ~1 GB VPS.
ENV NODE_OPTIONS=--max-old-space-size=768
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=deps /app/node_modules ./node_modules
COPY . .

RUN pnpm build

# ── runner ───────────────────────────────────────────────────────────────────
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0

# Standalone bundle: server.js + the minimal node_modules it needs.
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

USER node
EXPOSE 3000

CMD ["node", "server.js"]
