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

const normalizeWhitespace = (value: string) => value.replace(/\s+/g, " ").trim();

const escapeInline = (value: string) =>
  value.replace(/([\\`*_{}\[\]()#+\-.!>])/g, "\\$1");

const nodeToMarkdown = (node: Node): string => {
  if (node.nodeType === Node.TEXT_NODE) {
    return escapeInline((node.textContent || "").replace(/\u00a0/g, " "));
  }
  if (!(node instanceof HTMLElement)) return "";

  const tag = node.tagName.toLowerCase();
  const children = Array.from(node.childNodes).map(nodeToMarkdown).join("");
  const text = normalizeWhitespace(node.textContent || "");

  switch (tag) {
    case "br":
      return "  \n";
    case "p":
      return `${children.trim()}\n\n`;
    case "strong":
    case "b":
      return `**${children || text}**`;
    case "em":
    case "i":
      return `*${children || text}*`;
    case "code":
      if (node.closest("pre")) return children || text;
      return `\`${(children || text).replace(/`/g, "\\`")}\``;
    case "pre":
      return `\n\`\`\`\n${(node.textContent || "").trim()}\n\`\`\`\n\n`;
    case "h1":
      return `# ${children.trim()}\n\n`;
    case "h2":
      return `## ${children.trim()}\n\n`;
    case "h3":
      return `### ${children.trim()}\n\n`;
    case "h4":
      return `#### ${children.trim()}\n\n`;
    case "h5":
      return `##### ${children.trim()}\n\n`;
    case "h6":
      return `###### ${children.trim()}\n\n`;
    case "blockquote":
      return `${children
        .trim()
        .split("\n")
        .map((line) => (line.trim() ? `> ${line}` : ">"))
        .join("\n")}\n\n`;
    case "a": {
      const href = node.getAttribute("href") || "";
      const label = children.trim() || href;
      return href ? `[${label}](${href})` : label;
    }
    case "img": {
      const src = node.getAttribute("src") || "";
      if (!src) return "";
      const alt = node.getAttribute("alt") || "image";
      return `![${alt}](${src})`;
    }
    case "li":
      return `${children.trim()}\n`;
    case "ul":
      return `${Array.from(node.children)
        .map((li) => `- ${nodeToMarkdown(li).trim()}`)
        .join("\n")}\n\n`;
    case "ol":
      return `${Array.from(node.children)
        .map((li, i) => `${i + 1}. ${nodeToMarkdown(li).trim()}`)
        .join("\n")}\n\n`;
    case "hr":
      return `---\n\n`;
    default:
      return children;
  }
};

export const renderHtmlToMarkdown = (value?: string) => {
  const input = value ?? "";
  if (!input.trim()) return "";
  if (!looksLikeHtml(input)) return input;

  const doc = new DOMParser().parseFromString(input, "text/html");
  const markdown = Array.from(doc.body.childNodes).map(nodeToMarkdown).join("");
  return markdown.replace(/\n{3,}/g, "\n\n").trim();
};
