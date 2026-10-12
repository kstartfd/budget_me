import { z } from "zod"

/**
 * Auth feature schemas. Unlike the budget feature, auth talks to the real Go
 * backend from day one — these mirror api/dto on the Go side.
 */

export const userSchema = z.object({
  id: z.string(),
  first_name: z.string(),
  username: z.string().optional(),
  email: z.string().optional(),
  telegram_id: z.number().optional(),
  language_code: z.string().optional(),
})
export type User = z.infer<typeof userSchema>

export const authResponseSchema = z.object({
  token: z.string(),
  user: userSchema,
})
export type AuthResponse = z.infer<typeof authResponseSchema>

/** Shape validation for the web login form (RHF + zodResolver). */
export const loginFormSchema = z.object({
  email: z.string().trim().email("invalidEmail"),
  password: z.string().min(1, "passwordRequired"),
})
export type LoginFormValues = z.infer<typeof loginFormSchema>
