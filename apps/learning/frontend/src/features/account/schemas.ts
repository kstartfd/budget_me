import { z } from "zod"

export const linkEmailFormSchema = z.object({
  email: z.string().trim().email("invalidEmail"),
  password: z.string().min(8, "passwordShort"),
})
export type LinkEmailFormValues = z.infer<typeof linkEmailFormSchema>
