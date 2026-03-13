import { AdminCheckboxField, AdminTextField } from "./AriaControls";

type PublishFieldsProps = {
  publishedAt: string;
  published: boolean;
  onPublishedAtChange: (value: string) => void;
  onPublishedChange: (checked: boolean) => void;
};

export default function PublishFields({
  publishedAt,
  published,
  onPublishedAtChange,
  onPublishedChange,
}: PublishFieldsProps) {
  return (
    <>
      <AdminTextField
        label="Published at"
        type="datetime-local"
        value={publishedAt}
        onChange={onPublishedAtChange}
      />
      <AdminCheckboxField label="Published" checked={published} onChange={onPublishedChange} />
    </>
  );
}
