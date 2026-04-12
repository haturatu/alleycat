import type { RefObject } from "react";
import { AdminButton, AdminTextField } from "@cms/ui/AriaControls";

type TitleSlugFieldsProps = {
  title: string;
  slug: string;
  slugEditedManually: boolean;
  titleError?: string;
  slugError?: string;
  autoDisabled?: boolean;
  aiGenerateAvailable?: boolean;
  aiGenerateDisabled?: boolean;
  titleInputRef?: RefObject<HTMLInputElement | null>;
  slugInputRef?: RefObject<HTMLInputElement | null>;
  onTitleChange: (value: string) => void;
  onSlugChange: (value: string) => void;
  onAutoSlug: () => void;
  onAISlugGenerate?: () => void;
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
  aiGenerateAvailable,
  aiGenerateDisabled,
  titleInputRef,
  slugInputRef,
  onTitleChange,
  onSlugChange,
  onAutoSlug,
  onAISlugGenerate,
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
          {aiGenerateAvailable && onAISlugGenerate ? (
            <AdminButton
              type="button"
              className="admin-secondary admin-slug-action"
              disabled={aiGenerateDisabled}
              onPress={onAISlugGenerate}
            >
              AI Generate
            </AdminButton>
          ) : null}
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
