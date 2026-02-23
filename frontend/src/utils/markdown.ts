import MarkdownIt from "markdown-it";

const md = new MarkdownIt("default", {
  html: true,
  linkify: true,
  typographer: false,
  breaks: false,
});

export const renderMarkdownToHtml = (value?: string) => {
  const input = value ?? "";
  if (!input.trim()) return "";
  return md.render(input);
};

export const looksLikeHtml = (value?: string) => /<[a-z][^>]*>/i.test(value ?? "");
