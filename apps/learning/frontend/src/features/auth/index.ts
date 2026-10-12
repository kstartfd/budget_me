/**
 * Public surface of the auth feature.
 */

export { AuthProvider, useAuth, readToken } from "./auth-context"
export { AuthGate } from "./components/auth-gate"
export {
  ApiError,
  linkEmail,
} from "./api"
export type { AuthResponse, User } from "./schemas"
