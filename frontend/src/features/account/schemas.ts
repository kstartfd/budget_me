import { z } from "zod"

/**
 * Schemas for the «Аккаунт» settings section — linking a second login path onto
 * the current user so one `users` row has both an email/password and a Telegram
 * identity. Linking always happens from an authenticated session with proof of
 * ownership (a password, or the Mini App's HMAC-signed initData); a telegram_id
 * / @username is an identifier, never typed in as a secret.
 *
 * Shape validation only lives here (RHF + Zod at the HTTP edge). The real
 * ownership check — password hashing, Telegram initData verification, the
 * "already linked to another account" (409) conflict — is the backend's job.
 */

/** Web form: add an email + password login path. Password ≥ 8 (matches register). */
export const linkEmailFormSchema = z.object({
  email: z.string().trim().email("invalidEmail"),
  password: z.string().min(8, "passwordShort"),
})
export type LinkEmailFormValues = z.infer<typeof linkEmailFormSchema>
