export const validateTitle = (value: string) => (value.trim() ? undefined : "Title is required.");

export const validateSlug = (value: string) => {
  const trimmed = value.trim();
  if (!trimmed) return "Slug is required.";
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(trimmed)) {
    return "Use lowercase letters, numbers, and hyphens.";
  }
  return undefined;
};

export const validateBody = (value: string) => (value.trim() ? undefined : "Content is required.");

export const validateURL = (value: string) => {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  if (!trimmed.startsWith("/")) return "URL must start with '/'.";
  return undefined;
};
