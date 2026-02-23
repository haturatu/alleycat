import hljs from "highlight.js";
import { marked } from "marked";

marked.setOptions({
  gfm: true,
  breaks: true,
});

export const renderMarkdownToHtml = (value?: string) => {
  const input = value ?? "";
  if (!input.trim()) return "";

  const rendered = marked.parse(input) as string;
  const doc = new DOMParser().parseFromString(`<div id=\"md-root\">${rendered}</div>`, "text/html");
  const root = doc.getElementById("md-root");
  if (!root) return rendered;

  root.querySelectorAll("pre code").forEach((codeBlock) => {
    const classNames = (codeBlock.getAttribute("class") || "").split(/\s+/);
    const languageClass = classNames.find((className) => className.startsWith("language-"));
    const language = languageClass?.slice("language-".length);
    const source = codeBlock.textContent || "";

    if (language && hljs.getLanguage(language)) {
      codeBlock.innerHTML = hljs.highlight(source, { language }).value;
    } else {
      codeBlock.innerHTML = hljs.highlightAuto(source).value;
    }
    codeBlock.classList.add("hljs");
  });

  return root.innerHTML;
};

export const looksLikeHtml = (value?: string) => /<[a-z][^>]*>/i.test(value ?? "");

const normalizeText = (value: string) => value.replace(/\u00a0/g, " ");

const escapeText = (value: string) => value.replace(/([\\`*_{}#+!>])/g, "\\$1");

const nodeToMarkdown = (node: Node): string => {
  if (node.nodeType === Node.TEXT_NODE) {
    return escapeText(normalizeText(node.textContent || ""));
  }
  if (!(node instanceof HTMLElement)) return "";

  const tag = node.tagName.toLowerCase();
  const children = Array.from(node.childNodes).map(nodeToMarkdown).join("");

  switch (tag) {
    case "br":
      return "\n";
    case "p":
      return `${children.trim()}\n\n`;
    case "strong":
    case "b":
      return `**${children}**`;
    case "em":
    case "i":
      return `*${children}*`;
    case "code":
      if (node.closest("pre")) return children;
      return `\`${(node.textContent || "").replace(/`/g, "\\`")}\``;
    case "pre": {
      const code = (node.textContent || "").replace(/\n+$/g, "");
      return `\n\`\`\`\n${code}\n\`\`\`\n\n`;
    }
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
      return children.trim();
    case "ul": {
      const items = Array.from(node.children)
        .filter((child) => child.tagName.toLowerCase() === "li")
        .map((li) => `- ${nodeToMarkdown(li)}`)
        .join("\n");
      return `${items}\n\n`;
    }
    case "ol": {
      const items = Array.from(node.children)
        .filter((child) => child.tagName.toLowerCase() === "li")
        .map((li, i) => `${i + 1}. ${nodeToMarkdown(li)}`)
        .join("\n");
      return `${items}\n\n`;
    }
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
  return markdown
    .replace(/[ \t]+\n/g, "\n")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
};
