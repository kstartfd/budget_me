import createNextIntlPlugin from "next-intl/plugin"

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts")

/** @type {import('next').NextConfig} */
const nextConfig = {
  // Self-contained server bundle (.next/standalone/server.js) for the prod
  // Docker image — see infra/budget/docker/frontend.Dockerfile.
  output: "standalone",
  experimental: {
    optimizePackageImports: ["@tabler/icons-react"],
  },
  // Same-origin API: the app calls /api/v1/* and Next proxies it to the Go
  // backend, so one tunnel/domain serves both frontend and API.
  async rewrites() {
    const backend = process.env.BACKEND_URL ?? "http://localhost:8080"
    return [{ source: "/api/v1/:path*", destination: `${backend}/api/v1/:path*` }]
  },
}

export default withNextIntl(nextConfig)
