import fs from "node:fs";
import path from "node:path";
import { describe, expect, test } from "vitest";

import { renderHtmlToMarkdown, renderMarkdownToHtml } from "./markdown";

const fixturePath = path.resolve(__dirname, "../../../testdata/markdown-css-test-cases.md");

type MarkdownCase = {
  name: string;
  markdown: string;
};

const normalizeMarkdown = (value: string) => value.replace(/\r\n?/g, "\n").trim();

const normalizeHtml = (value: string) => {
  const doc = new DOMParser().parseFromString(`<div id="root">${value}</div>`, "text/html");
  const root = doc.getElementById("root");
  return (root?.innerHTML || "")
    .replace(/\r\n?/g, "\n")
    .replace(/>\s+</g, "><")
    .trim();
};

const loadMarkdownCases = (): MarkdownCase[] => {
  const source = fs.readFileSync(fixturePath, "utf8").replace(/\r\n?/g, "\n").trim();
  const cases: MarkdownCase[] = [{ name: "whole document", markdown: source }];

  const parts = source.split(/\n(?=## )/g);
  for (const part of parts) {
    if (!part.startsWith("## ")) continue;
    const firstLine = part.split("\n", 1)[0].replace(/^##\s+/, "").trim();
    cases.push({
      name: firstLine || `section ${cases.length}`,
      markdown: part.trim(),
    });
  }

  return cases;
};

describe("markdown round-trip", () => {
  const cases = loadMarkdownCases();

  test.each(cases)("$name stays stable across markdown/html round-trips", ({ markdown }) => {
    const html1 = normalizeHtml(renderMarkdownToHtml(markdown, { highlightCode: false }));
    const markdown1 = normalizeMarkdown(renderHtmlToMarkdown(html1));
    const html2 = normalizeHtml(renderMarkdownToHtml(markdown1, { highlightCode: false }));
    const markdown2 = normalizeMarkdown(renderHtmlToMarkdown(html2));

    expect(html2).toBe(html1);
    expect(markdown2).toBe(markdown1);
  });
});
