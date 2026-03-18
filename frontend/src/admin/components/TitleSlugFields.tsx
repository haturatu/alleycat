import type { RefObject } from "react";
import { AdminButton, AdminTextField } from "./AriaControls";

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
}: TitleSlugFieldsProps) {
  return (
    <>
      <AdminTextField inputRef={titleInputRef} label="Title" value={title} onChange={onTitleChange} />
      {titleError && <p className="admin-error-inline">{titleError}</p>}
      <div className="admin-field">
        <span>Slug</span>
        <div className="admin-inline">
          <AdminTextField
            ariaLabel="Slug"
            label=""
            inputRef={slugInputRef}
            value={slug}
            onChange={onSlugChange}
            className="admin-field"
          />
          <AdminButton type="button" disabled={autoDisabled} onPress={onAutoSlug}>
            Generate Slug
          </AdminButton>
        </div>
      </div>
      {slugError ? (
        <p className="admin-error-inline">{slugError}</p>
      ) : (
        <p className="admin-note">
          {slugEditedManually ? "Slug is locked (manual)." : "Slug follows title automatically."}
        </p>
      )}
    </>
  );
}
