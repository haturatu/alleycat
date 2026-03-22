import hljs from "highlight.js";
import { marked } from "marked";

marked.setOptions({
  gfm: true,
  breaks: true,
});

type RenderMarkdownOptions = {
  highlightCode?: boolean;
};

const alertKinds = new Set(["note", "tip", "important", "warning", "caution"]);

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

const alertMarkerRe = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]$/i;

const extractAlertKind = (value: string) => {
  const match = value.trim().match(alertMarkerRe);
  if (!match) return null;
  const kind = match[1].toLowerCase();
  return alertKinds.has(kind) ? kind : null;
};

const splitAlertParagraph = (element: HTMLElement, doc: Document) => {
  const firstChild = element.firstChild;
  if (!(firstChild instanceof Text)) return null;

  const raw = firstChild.textContent || "";
  const lines = raw.replace(/\r\n?/g, "\n").split("\n");
  const marker = lines[0]?.trim() || "";
  const kind = extractAlertKind(marker);
  if (!kind) return null;

  const remainder = lines.slice(1).join("\n").trim();
  const contentNodes: Node[] = [];

  if (remainder) {
    contentNodes.push(doc.createTextNode(remainder));
  }

  let consumeLeadingBreak = Boolean(remainder) || raw.includes("\n");
  Array.from(element.childNodes).slice(1).forEach((node) => {
    if (consumeLeadingBreak && node instanceof HTMLBRElement) {
      consumeLeadingBreak = false;
      return;
    }
    contentNodes.push(node.cloneNode(true));
  });

  const hasContent = contentNodes.some((node) => (node.textContent || "").trim() !== "");
  if (!hasContent) {
    return { kind, contentNodes: [] as Node[] };
  }

  const paragraph = doc.createElement("p");
  contentNodes.forEach((node) => {
    paragraph.appendChild(node);
  });

  return {
    kind,
    contentNodes: [paragraph],
  };
};

const createAlertNode = (doc: Document, kind: string, contentNodes: Node[]) => {
  const wrapper = doc.createElement("div");
  wrapper.className = `markdown-alert markdown-alert-${kind}`;

  const title = doc.createElement("p");
  title.className = "markdown-alert-title";
  title.textContent = kind.charAt(0).toUpperCase() + kind.slice(1);
  wrapper.appendChild(title);

  contentNodes.forEach((node) => {
    wrapper.appendChild(node);
  });

  return wrapper;
};

const normalizeMarkdownAlertsInContainer = (container: ParentNode, doc: Document) => {
  Array.from(container.children).forEach((element) => {
    if (element instanceof HTMLElement && element.tagName === "BLOCKQUOTE") return;
    normalizeMarkdownAlertsInContainer(element, doc);
  });

  Array.from(container.children).forEach((element) => {
    if (!(element instanceof HTMLElement) || element.tagName !== "BLOCKQUOTE") return;
    const first = element.firstElementChild;
    if (!(first instanceof HTMLElement) || first.tagName !== "P") return;
    const split = splitAlertParagraph(first, doc);
    if (!split) return;

    const contentNodes = [
      ...split.contentNodes,
      ...Array.from(element.children)
        .filter((child) => child !== first)
        .map((node) => node.cloneNode(true)),
    ];
    const replacement = createAlertNode(doc, split.kind, contentNodes);
    element.replaceWith(replacement);
  });

  const children = Array.from(container.children);
  for (let index = 0; index < children.length; index += 1) {
    const element = children[index];
    if (!(element instanceof HTMLElement) || element.tagName !== "P") continue;
    const split = splitAlertParagraph(element, doc);
    if (!split) continue;

    const contentNodes: Node[] = [...split.contentNodes];
    let cursor = element.nextSibling;
    while (cursor) {
      const next = cursor.nextSibling;
      if (!(cursor instanceof HTMLElement)) {
        cursor = next;
        continue;
      }
      if (cursor.tagName === "P" && splitAlertParagraph(cursor, doc)) break;
      if (/^H[1-6]$/.test(cursor.tagName) || cursor.tagName === "HR") break;
      contentNodes.push(cursor);
      cursor = next;
    }

    const replacement = createAlertNode(doc, split.kind, contentNodes);
    element.replaceWith(replacement);
    contentNodes.forEach((node) => {
      if (node.parentNode) {
        node.parentNode.removeChild(node);
      }
    });
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
  normalizeMarkdownAlertsInContainer(root, doc);
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

const renderInlineChildrenMarkdown = (node: ParentNode) =>
  Array.from(node.childNodes)
    .map(nodeToMarkdown)
    .join("");

const renderAlertMarkdown = (kind: string, content: string) => {
  const lines = content.trim().split("\n");
  const quoted = [`> [!${kind.toUpperCase()}]`];

  lines.forEach((line) => {
    quoted.push(line.trim() ? `> ${line}` : ">");
  });

  return `${quoted.join("\n")}\n\n`;
};

const tableAlignmentMarker = (cell: HTMLElement) => {
  const align = (cell.getAttribute("align") || "").toLowerCase();
  switch (align) {
    case "right":
      return "---:";
    case "center":
      return ":---:";
    case "left":
      return ":---";
    default:
      return "---";
  }
};

const renderTableMarkdown = (table: HTMLElement) => {
  const rows = Array.from(table.querySelectorAll("tr"));
  if (rows.length === 0) return "";

  const headerCells = Array.from(rows[0].children).filter(
    (cell): cell is HTMLElement => cell instanceof HTMLElement && /^(TH|TD)$/.test(cell.tagName)
  );
  if (headerCells.length === 0) return "";

  const renderCell = (cell: HTMLElement) => renderInlineChildrenMarkdown(cell).replace(/\n+/g, " ").trim();
  const header = `| ${headerCells.map(renderCell).join(" | ")} |`;
  const separator = `| ${headerCells.map(tableAlignmentMarker).join(" | ")} |`;
  const body = rows
    .slice(1)
    .map((row) => {
      const cells = Array.from(row.children).filter(
        (cell): cell is HTMLElement => cell instanceof HTMLElement && /^(TH|TD)$/.test(cell.tagName)
      );
      if (cells.length === 0) return "";
      return `| ${cells.map(renderCell).join(" | ")} |`;
    })
    .filter(Boolean);

  return `${[header, separator, ...body].join("\n")}\n\n`;
};

const alertKindFromElement = (element: HTMLElement) => {
  const className = element.className || "";
  const match = className.match(/markdown-alert-(note|tip|important|warning|caution)\b/i);
  return match ? match[1].toLowerCase() : null;
};

const renderListItemMarkdown = (li: HTMLElement, ordered: boolean, index: number, depth: number): string => {
  const indent = "  ".repeat(depth);
  const marker = ordered ? `${index + 1}. ` : "- ";

  const inlineParts: string[] = [];
  const nestedParts: string[] = [];

  Array.from(li.childNodes).forEach((child) => {
    if (child instanceof HTMLElement) {
      const tag = child.tagName.toLowerCase();
      if (tag === "ul") {
        const nestedIndent = `${indent}${ordered ? "   " : "  "}`;
        nestedParts.push(
          renderListMarkdown(child, false, 0)
            .split("\n")
            .map((line) => (line ? `${nestedIndent}${line}` : line))
            .join("\n")
        );
        return;
      }
      if (tag === "ol") {
        const nestedIndent = `${indent}${ordered ? "   " : "  "}`;
        nestedParts.push(
          renderListMarkdown(child, true, 0)
            .split("\n")
            .map((line) => (line ? `${nestedIndent}${line}` : line))
            .join("\n")
        );
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
    case "del":
    case "s":
    case "strike":
      return `~~${children}~~`;
    case "mark":
      return `<mark>${children}</mark>`;
    case "code":
      if (node.closest("pre")) return children;
      return `\`${(node.textContent || "").replace(/`/g, "\\`")}\``;
    case "input":
      if (node.getAttribute("type")?.toLowerCase() !== "checkbox") return "";
      return node.hasAttribute("checked") ? "[x]" : "[ ]";
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
    case "blockquote": {
      const first = node.firstElementChild;
      if (first instanceof HTMLElement && first.tagName === "P") {
        const split = splitAlertParagraph(first, node.ownerDocument);
        if (split) {
          const contentMarkdown = [split.contentNodes.map(nodeToMarkdown).join(""), ...Array.from(node.children).slice(1).map(nodeToMarkdown)]
            .join("")
            .replace(/\n{3,}/g, "\n\n")
            .trim();
          return renderAlertMarkdown(split.kind, contentMarkdown);
        }
      }
      return `${children
        .trim()
        .split("\n")
        .map((line) => (line.trim() ? `> ${line}` : ">"))
        .join("\n")}\n\n`;
    }
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
    case "table":
      return renderTableMarkdown(node);
    case "div": {
      const kind = alertKindFromElement(node);
      if (kind) {
        const contentMarkdown = Array.from(node.children)
          .filter((child) => !(child instanceof HTMLElement && child.classList.contains("markdown-alert-title")))
          .map(nodeToMarkdown)
          .join("")
          .replace(/\n{3,}/g, "\n\n")
          .trim();
        return renderAlertMarkdown(kind, contentMarkdown);
      }
      return `${node.outerHTML.trim()}\n\n`;
    }
    default:
      return children;
  }
};

export const renderHtmlToMarkdown = (value?: string) => {
  const input = value ?? "";
  if (!input.trim()) return "";

  const doc = new DOMParser().parseFromString(input, "text/html");
  normalizeFencedCodeBlocksInContainer(doc.body, doc);

  return Array.from(doc.body.childNodes)
    .map(nodeToMarkdown)
    .join("")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
};

export const normalizeFencedCodeBlocksInHtml = (value?: string) => {
  const input = value ?? "";
  if (!input.trim()) return "";

  const doc = new DOMParser().parseFromString(input, "text/html");
  normalizeFencedCodeBlocksInContainer(doc.body, doc);
  return doc.body.innerHTML;
};
