import hljs from "highlight.js";
import { marked } from "marked";

marked.setOptions({
  gfm: true,
  breaks: true,
});

type RenderMarkdownOptions = {
  highlightCode?: boolean;
};

const codeFenceStartRe = /^```([\w+-]+)?\s*$/;
const codeFenceEndRe = /^```\s*$/;

const highlightCodeBlocks = (root: ParentNode, highlightCode: boolean) => {
  root.querySelectorAll("pre code").forEach((codeBlock) => {
    const classNames = (codeBlock.getAttribute("class") || "").split(/\s+/);
    const languageClass = classNames.find((className) => className.startsWith("language-"));
    const language = languageClass?.slice("language-".length);
    const source = (codeBlock.textContent || "").replace(/\r\n?/g, "\n").replace(/\n$/, "");

    if (!highlightCode) {
      codeBlock.textContent = source;
      return;
    }

    if (language && hljs.getLanguage(language)) {
      codeBlock.innerHTML = hljs.highlight(source, { language }).value;
    } else {
      codeBlock.innerHTML = hljs.highlightAuto(source).value;
    }
    codeBlock.classList.add("hljs");
  });
};

const isStandaloneFenceNode = (node: Node) => {
  if (node.nodeType === Node.TEXT_NODE) {
    return Boolean((node.textContent || "").trim());
  }
  if (!(node instanceof HTMLElement)) return false;
  if (node.tagName === "PRE" || node.tagName === "CODE") return false;
  return Array.from(node.children).every((child) => child.tagName === "BR");
};

const getFenceLine = (node: Node) => {
  if (!isStandaloneFenceNode(node)) return null;
  return (node.textContent || "").replace(/\u00a0/g, " ").trim();
};

const normalizeFencedCodeBlocksInContainer = (container: ParentNode, doc: Document) => {
  const nodes = Array.from(container.childNodes);
  for (let index = 0; index < nodes.length; index += 1) {
    const startNode = nodes[index];
    if (startNode instanceof HTMLElement && (startNode.tagName === "PRE" || startNode.tagName === "CODE")) {
      continue;
    }

    const startLine = getFenceLine(startNode);
    const startMatch = startLine?.match(codeFenceStartRe);
    if (!startMatch) {
      if (startNode instanceof HTMLElement) {
        normalizeFencedCodeBlocksInContainer(startNode, doc);
      }
      continue;
    }

    let endIndex = -1;
    for (let cursor = index + 1; cursor < nodes.length; cursor += 1) {
      const candidate = nodes[cursor];
      const line = getFenceLine(candidate);
      if (line && codeFenceEndRe.test(line)) {
        endIndex = cursor;
        break;
      }
    }
    if (endIndex === -1) continue;

    const code = nodes
      .slice(index + 1, endIndex)
      .map((node) => (node.textContent || "").replace(/\r\n?/g, "\n"))
      .join("\n")
      .replace(/\n+$/g, "");

    const pre = doc.createElement("pre");
    const codeElement = doc.createElement("code");
    const language = startMatch[1]?.trim();
    if (language) {
      codeElement.className = `language-${language}`;
    }
    codeElement.textContent = code;
    pre.appendChild(codeElement);

    const firstNode = nodes[index];
    firstNode.parentNode?.insertBefore(pre, firstNode);
    nodes.slice(index, endIndex + 1).forEach((node) => node.parentNode?.removeChild(node));
    index = endIndex;
  }
};

export const renderMarkdownToHtml = (value?: string, options: RenderMarkdownOptions = {}) => {
  const input = value ?? "";
  if (!input.trim()) return "";
  const { highlightCode = true } = options;

  const rendered = marked.parse(input) as string;
  const doc = new DOMParser().parseFromString(`<div id=\"md-root\">${rendered}</div>`, "text/html");
  const root = doc.getElementById("md-root");
  if (!root) return rendered;
  highlightCodeBlocks(root, highlightCode);

  return root.innerHTML;
};

const htmlTagRe = /<\s*\/?\s*([a-z][a-z0-9-]*)\b[^>]*>/i;
const htmlLikeTags = new Set([
  "p",
  "div",
  "span",
  "a",
  "img",
  "ul",
  "ol",
  "li",
  "pre",
  "code",
  "blockquote",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "table",
  "thead",
  "tbody",
  "tr",
  "th",
  "td",
  "hr",
  "br",
]);

export const looksLikeHtml = (value?: string) => {
  const input = value ?? "";
  if (!input.trim()) return false;
  const match = input.match(htmlTagRe);
  if (!match) return false;
  return htmlLikeTags.has(match[1].toLowerCase());
};

const normalizeText = (value: string) => value.replace(/\u00a0/g, " ");

const escapeText = (value: string) => value.replace(/([\\`*_{}#+!>])/g, "\\$1");

const renderListItemMarkdown = (li: HTMLElement, ordered: boolean, index: number, depth: number): string => {
  const indent = "  ".repeat(depth);
  const marker = ordered ? `${index + 1}. ` : "- ";

  const inlineParts: string[] = [];
  const nestedParts: string[] = [];

  Array.from(li.childNodes).forEach((child) => {
    if (child instanceof HTMLElement) {
      const tag = child.tagName.toLowerCase();
      if (tag === "ul") {
        nestedParts.push(renderListMarkdown(child, false, depth + 1));
        return;
      }
      if (tag === "ol") {
        nestedParts.push(renderListMarkdown(child, true, depth + 1));
        return;
      }
    }
    inlineParts.push(nodeToMarkdown(child));
  });

  const inlineText = inlineParts.join("").replace(/\n+/g, " ").trim();
  const head = `${indent}${marker}${inlineText}`.trimEnd();
  if (nestedParts.length === 0) return head;

  const nested = nestedParts.filter(Boolean).join("\n");
  return nested ? `${head}\n${nested}` : head;
};

const renderListMarkdown = (list: HTMLElement, ordered: boolean, depth: number): string => {
  return Array.from(list.children)
    .filter((child): child is HTMLElement => child.tagName.toLowerCase() === "li")
    .map((li, index) => renderListItemMarkdown(li, ordered, index, depth))
    .join("\n");
};

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
      const codeNode = node.querySelector("code");
      const classNames = `${node.getAttribute("class") || ""} ${codeNode?.getAttribute("class") || ""}`;
      const langClass = classNames.split(/\s+/).find((className) => className.startsWith("language-"));
      const language = langClass ? langClass.slice("language-".length) : "";
      const source = (codeNode?.textContent || node.textContent || "")
        .replace(/\r\n?/g, "\n")
        .replace(/\n+$/g, "");
      const fence = language ? `\`\`\`${language}` : "```";
      return `\n${fence}\n${source}\n\`\`\`\n\n`;
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
      return renderListItemMarkdown(node, false, 0, 0);
    case "ul":
      return `${renderListMarkdown(node, false, 0)}\n\n`;
    case "ol":
      return `${renderListMarkdown(node, true, 0)}\n\n`;
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

export const normalizeFencedCodeBlocksInHtml = (value?: string, options: RenderMarkdownOptions = {}) => {
  const input = value ?? "";
  if (!input.trim()) return "";

  const { highlightCode = true } = options;
  const doc = new DOMParser().parseFromString(`<div id="html-root">${input}</div>`, "text/html");
  const root = doc.getElementById("html-root");
  if (!root) return input;

  normalizeFencedCodeBlocksInContainer(root, doc);
  highlightCodeBlocks(root, highlightCode);
  return root.innerHTML;
};

export const renderStoredContentToHtml = (value?: string, options: RenderMarkdownOptions = {}) => {
  const input = value ?? "";
  if (!input.trim()) return "";
  return looksLikeHtml(input)
    ? normalizeFencedCodeBlocksInHtml(input, options)
    : renderMarkdownToHtml(input, options);
};
