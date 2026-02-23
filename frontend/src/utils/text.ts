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

const escapeHtml = (value: string) =>
  value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");

export const normalizeMarkdownLinksInHtml = (value?: string) => {
  const input = value ?? "";
  if (!input) return input;

  // Convert markdown links in text nodes while keeping existing HTML tags untouched.
  const markdownLinkRe = /\[([^\]\n]+)\]\((https?:\/\/[^\s)]+)\)/g;
  return input
    .split(/(<[^>]+>)/g)
    .map((part) => {
      if (part.startsWith("<") && part.endsWith(">")) return part;
      return part.replace(markdownLinkRe, (_full, label: string, href: string) => {
        const safeHref = href.trim();
        if (!/^https?:\/\//i.test(safeHref)) return _full;
        return `<a href="${escapeHtml(safeHref)}" target="_blank" rel="noopener noreferrer">${escapeHtml(
          label.trim()
        )}</a>`;
      });
    })
    .join("");
};
