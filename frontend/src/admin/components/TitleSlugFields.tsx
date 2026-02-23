type TitleSlugFieldsProps = {
  title: string;
  slug: string;
  slugEditedManually: boolean;
  titleError?: string;
  slugError?: string;
  autoDisabled?: boolean;
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
  onTitleChange,
  onSlugChange,
  onAutoSlug,
}: TitleSlugFieldsProps) {
  return (
    <>
      <label>
        Title
        <input value={title} onChange={(e) => onTitleChange(e.target.value)} />
      </label>
      {titleError && <p className="admin-error-inline">{titleError}</p>}
      <label>
        Slug
        <div className="admin-inline">
          <input value={slug} onChange={(e) => onSlugChange(e.target.value)} />
          <button type="button" disabled={autoDisabled} onClick={onAutoSlug}>
            Auto
          </button>
        </div>
      </label>
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
