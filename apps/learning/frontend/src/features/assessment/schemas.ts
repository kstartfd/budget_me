import { z } from "zod"

const rangeSchema = z.object({
  lower: z.string(),
  upper: z.string(),
  boundary: z.string(),
})

const knowledgeSchema = z.object({
  knowledgeItemKey: z.string(),
  category: z.string(),
  learningObjective: z.string(),
  status: z.enum([
    "demonstrated",
    "needs_practice",
    "needs_more_evidence",
    "not_assessed",
  ]),
  gapSupport: z.string().optional(),
  evidenceIds: z.array(z.string()),
})

const resultSchema = z.object({
  difficultyRange: rangeSchema.nullable(),
  evidenceSufficiency: z.enum([
    "insufficient",
    "preliminary",
    "sufficient_for_screening",
  ]),
  conflictingEvidence: z.boolean(),
  skillEstimates: z.array(
    z.object({
      skill: z.string(),
      difficultyRange: rangeSchema.nullable(),
      usableEvidence: z.number(),
    }),
  ),
  sections: z
    .array(
      z.object({
        skill: z.string(),
        status: z.enum(["completed", "partial", "failed", "cancelled"]),
        stopReason: z.string(),
        evidenceSufficiency: z.string(),
        difficultyRange: rangeSchema.nullable(),
        usableEvidence: z.number(),
        evidenceIds: z.array(z.string()),
      }),
    )
    .optional(),
  unassessedSkills: z.array(z.string()),
  knowledgeOutcomes: z.array(knowledgeSchema),
  demonstratedStrengths: z.array(knowledgeSchema),
  knowledgeGaps: z.array(knowledgeSchema),
  recommendedKnowledgeChecks: z.array(knowledgeSchema),
  recommendedLearningObjectives: z.array(knowledgeSchema),
})

export const assessmentProgressSchema = z.object({
  type: z.literal("quick_screening"),
  status: z.enum([
    "created",
    "in_progress",
    "paused",
    "evaluating",
    "completed",
    "partial",
    "failed",
    "cancelled",
  ]),
  learnerProfileId: z.string(),
  answered: z.number(),
  maxQuestions: z.number(),
  stopReason: z.string().optional(),
  // Older servers do not return section storage; the legacy fields remain valid.
  sections: z.array(z.object({
    id: z.string(),
    skill: z.string(),
    position: z.number(),
    status: z.enum(["created", "in_progress", "paused", "evaluating", "completed", "partial", "failed", "cancelled"]),
    currentBand: z.string(),
    batch: z.number(),
    answered: z.number(),
    maxQuestions: z.number(),
    stopReason: z.string().optional(),
  })).optional(),
  result: resultSchema.nullish(),
})

export const assessmentQuestionTypeSchema = z.enum([
  "multiple_choice",
  "open_gap_fill",
  "error_correction",
  "sentence_transformation",
  "vocabulary_in_context",
  "reading_comprehension",
])

export type AssessmentResult = z.infer<typeof resultSchema>
export type AssessmentRange = z.infer<typeof rangeSchema>
export type KnowledgeOutcome = z.infer<typeof knowledgeSchema>
