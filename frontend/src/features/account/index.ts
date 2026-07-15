/**
 * Public surface of the account feature — the «Аккаунт» settings section that
 * links a second login path (email/password ↔ Telegram) onto the current user.
 * App routes import the section only; the hooks/schemas are exposed for tests
 * and for the eventual backend swap.
 */

export { AccountSection } from "./components/account-section"
export { DeleteAccountSection } from "./components/delete-account-section"
export { DeletionBanner } from "./components/deletion-banner"
export { useAccount, type AccountIdentities } from "./queries"
export {
  useLinkEmail,
  useRequestAccountDeletion,
  useCancelAccountDeletion,
} from "./mutations"
export type { LinkEmailFormValues } from "./schemas"
