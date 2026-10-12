import { QueryClient } from "@tanstack/react-query"

/**
 * Builds the app-wide React Query client. A single instance lives in the
 * root providers (see app/providers.tsx) so its cache persists across client
 * navigations — repeated per-page reads of the same server data collapse
 * into one request each.
 *
 * Defaults:
 *  - staleTime 30s: server data is reused without a refetch within the window;
 *    mutations (create/edit/delete) invalidate explicitly. Dictionaries
 *    override this with a much longer staleTime at the call site.
 *  - refetchOnWindowFocus off: navigation/tab-switching shouldn't spam the API.
 *  - retry 1: one transient-failure retry; 401s are handled by the auth layer.
 */
export function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        refetchOnWindowFocus: false,
        retry: 1,
      },
    },
  })
}
