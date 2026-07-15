"use client"

import { useTranslations } from "next-intl"

import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog"
import {
  BottomSheet,
  isDuplicateNameError,
  useBudgetUI,
  useCreateCategory,
  useDeleteCategory,
  useUpdateCategory,
  type Category,
  type Direction,
} from "@/features/budget"
import { useIsDesktop } from "@/hooks/use-is-desktop"

import { CategoryForm } from "../forms/category-form"
import { DEFAULT_CATEGORY_COLOR } from "../palette"
import type { CategoryFormValues } from "../schemas"

/** What the categories screen asks the sheet to do: add (no category) or edit. */
export interface CategorySheetRequest {
  kind: Direction
  /** Present → edit this category; absent → create a new one. */
  category?: Category
}

/**
 * Add/edit category surface: a bottom sheet on mobile, a centered dialog on
 * desktop (design "add sheet" / "edit modal"), same CategoryForm inside both.
 * Owns the create/update/delete mutations and the success toasts; the parent
 * mounts it only while a request is open (and keys it, so the form re-inits).
 */
export function CategorySheet({
  request,
  categoryName,
  onClose,
}: {
  request: CategorySheetRequest
  /** Resolved display name of the edited category ("" when creating). */
  categoryName: string
  onClose: () => void
}) {
  const isDesktop = useIsDesktop()
  const t = useTranslations("categoriesPage")
  const { showToast } = useBudgetUI()

  const create = useCreateCategory()
  const update = useUpdateCategory()
  const remove = useDeleteCategory()

  const editing = request.category != null
  const title = editing ? t("editTitle") : t("newTitle")

  const defaultValues: CategoryFormValues = {
    name: editing ? categoryName : "",
    color: request.category?.color ?? DEFAULT_CATEGORY_COLOR,
  }

  function handleSubmit(values: CategoryFormValues) {
    if (editing && request.category) {
      update.mutate(
        {
          id: request.category.id,
          kind: request.kind,
          name: values.name,
          color: values.color,
        },
        {
          onSuccess: () => {
            onClose()
            showToast("success", t("updated", { name: values.name }))
          },
        },
      )
      return
    }
    create.mutate(
      { kind: request.kind, name: values.name, color: values.color },
      {
        onSuccess: () => {
          onClose()
          showToast("success", t("created", { name: values.name }))
        },
        onError: (error) => {
          // Duplicate personal name (backend 409 / code 40001): keep the sheet
          // open so the user can rename. Other errors bubble to the default.
          if (isDuplicateNameError(error)) {
            showToast("danger", t("nameTaken"))
          }
        },
      },
    )
  }

  function handleDelete() {
    if (!request.category) return
    remove.mutate(
      { id: request.category.id, kind: request.kind },
      {
        onSuccess: () => {
          onClose()
          showToast("info", t("deleted", { name: categoryName }))
        },
      },
    )
  }

  const form = (
    <CategoryForm
      mode={editing ? "edit" : "create"}
      defaultValues={defaultValues}
      submitting={create.isPending || update.isPending}
      deleting={remove.isPending}
      onSubmit={handleSubmit}
      onDelete={editing ? handleDelete : undefined}
      onCancel={onClose}
    />
  )

  if (isDesktop) {
    return (
      <Dialog open onOpenChange={(open) => !open && onClose()}>
        <DialogContent
          aria-describedby={undefined}
          className="bg-card border-border gap-0 rounded-[26px] p-6 sm:max-w-[420px]"
        >
          <DialogTitle className="text-lg font-bold tracking-[-0.02em]">
            {title}
          </DialogTitle>
          {form}
        </DialogContent>
      </Dialog>
    )
  }

  return (
    <BottomSheet title={title} onClose={onClose}>
      {form}
    </BottomSheet>
  )
}
