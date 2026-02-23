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
      <label>
        Published at
        <input
          type="datetime-local"
          value={publishedAt}
          onChange={(e) => onPublishedAtChange(e.target.value)}
        />
      </label>
      <label className="admin-check admin-check-right">
        <span>Published</span>
        <input
          type="checkbox"
          checked={published}
          onChange={(e) => onPublishedChange(e.target.checked)}
        />
      </label>
    </>
  );
}
