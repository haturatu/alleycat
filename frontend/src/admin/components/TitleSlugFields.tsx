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
  onToggleSlugMode?: () => void;
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
}: TitleSlugFieldsProps) {
  return (
    <>
      <AdminTextField inputRef={titleInputRef} label="Title" value={title} onChange={onTitleChange} />
      {titleError && <p className="admin-error-inline">{titleError}</p>}
      <div className="admin-field">
        <div className="admin-field-head">
          <span>Slug</span>
          {onToggleSlugMode ? (
            <AdminButton type="button" className="admin-secondary" onPress={onToggleSlugMode}>
              {slugEditedManually ? "Use Auto Slug" : "Lock Slug"}
            </AdminButton>
          ) : null}
        </div>
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
          {slugEditedManually ? "Manual mode is on. You can switch back to automatic generation." : "Slug follows title automatically until you lock it."}
        </p>
      )}
    </>
  );
}
