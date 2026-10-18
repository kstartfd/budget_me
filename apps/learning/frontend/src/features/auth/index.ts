/**
 * Public surface of the auth feature.
 */

export { AuthProvider, useAuth, readToken } from "./auth-context"
export { AuthGate } from "./components/auth-gate"
export {
  ApiError,
  cancelAccountDeletion,
  linkEmail,
  requestAccountDeletion,
} from "./api"
export type { AuthResponse, User } from "./schemas"
