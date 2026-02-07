export const slugify = (value: string) =>
  value
    .toLowerCase()
    .trim()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9\-]/g, "")
    .replace(/--+/g, "-")
    .replace(/^-+|-+$/g, "");

export const stripHtml = (value?: string) => {
  const safe = value ?? "";
  return safe.replace(/<[^>]*>/g, " ").replace(/\s+/g, " ").trim();
};

export const readingTimeMinutes = (value?: string) => {
  const text = stripHtml(value);
  if (!text) return 1;
  const charCount = text.length;
  return Math.max(1, Math.ceil(charCount / 700));
};

export const formatDate = (value?: string) => {
  if (!value) return "";
  const date = new Date(value);
  return date.toLocaleDateString("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
};

export const buildExcerpt = (value?: string, length = 160) => {
  const text = stripHtml(value);
  if (text.length <= length) return text;
  return `${text.slice(0, length)}...`;
};

export const parseTags = (value?: string) =>
  value
    ? value
        .split(",")
        .map((tag) => tag.trim())
        .filter(Boolean)
    : [];
