type UsePublishStateParams = {
  setPublishedAt: (value: string) => void;
  setPublished: (value: boolean) => void;
  markDirty: () => void;
};

export default function usePublishState({
  setPublishedAt,
  setPublished,
  markDirty,
}: UsePublishStateParams) {
  const onPublishedAtChange = (value: string) => {
    setPublishedAt(value);
    markDirty();
  };

  const onPublishedChange = (checked: boolean) => {
    setPublished(checked);
    markDirty();
  };

  return {
    onPublishedAtChange,
    onPublishedChange,
  };
}
