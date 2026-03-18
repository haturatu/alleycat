import type { RefObject } from "react";
import { AdminButton, AdminTextField } from "@cms/ui/AriaControls";

type TitleSlugFieldsProps = {
  title: string;
  slug: string;
  slugEditedManually: boolean;
  titleError?: string;
  slugError?: string;
  autoDisabled?: boolean;
  titleInputRef?: RefObject<HTMLInputElement>;
  slugInputRef?: RefObject<HTMLInputElement>;
  onTitleChange: (value: string) => void;
  onSlugChange: (value: string) => void;
  onAutoSlug: () => void;
  onToggleSlugMode?: () => void;
  editorial?: boolean;
};

export default function TitleSlugFields({
  title,
  slug,
  slugEditedManually,
  titleError,
  slugError,
  autoDisabled,
  titleInputRef,
  slugInputRef,
  onTitleChange,
  onSlugChange,
  onAutoSlug,
  onToggleSlugMode,
  editorial = false,
}: TitleSlugFieldsProps) {
  return (
    <>
      <AdminTextField
        inputRef={titleInputRef}
        inputClassName={editorial ? "admin-title-input" : "admin-input"}
        label={editorial ? "" : "Title"}
        ariaLabel="Title"
        value={title}
        onChange={onTitleChange}
        placeholder={editorial ? "Untitled" : undefined}
      />
      {titleError && <p className="admin-error-inline">{titleError}</p>}
      <div className={`admin-field ${editorial ? "admin-slug-field" : ""}`}>
        <div className="admin-field-head">
          <span>Slug</span>
        </div>
        <div className={`admin-inline ${editorial ? "admin-slug-inline" : ""}`}>
          <AdminTextField
            ariaLabel="Slug"
            label=""
            inputRef={slugInputRef}
            inputClassName={editorial ? "admin-slug-input" : "admin-input"}
            value={slug}
            onChange={onSlugChange}
            className="admin-field"
          />
          {onToggleSlugMode ? (
            <AdminButton type="button" className="admin-secondary admin-slug-action" onPress={onToggleSlugMode}>
              {slugEditedManually ? "Auto" : "Manual"}
            </AdminButton>
          ) : null}
          <AdminButton type="button" className="admin-secondary admin-slug-action" disabled={autoDisabled} onPress={onAutoSlug}>
            Generate
          </AdminButton>
        </div>
      </div>
      {slugError ? (
        <p className="admin-error-inline">{slugError}</p>
      ) : (
        <p className="admin-note">
          {slugEditedManually ? "Manual mode is on." : "Slug follows title automatically."}
        </p>
      )}
    </>
  );
}
