import { useRef, type ClipboardEvent } from "react";
import { renderHtmlToMarkdown, renderMarkdownToHtml } from "../../utils/markdown";
import { uploadImageAndGetURL } from "../mediaUpload";
import RichEditor from "../RichEditor";

export type EditorMode = "rich" | "markdown";
export type MarkdownViewMode = "write" | "preview";

type ContentEditorFieldProps = {
  body: string;
  markdownBody: string;
  editorMode: EditorMode;
  markdownViewMode: MarkdownViewMode;
  onBodyChange: (value: string) => void;
  onMarkdownBodyChange: (value: string) => void;
  onEditorModeChange: (mode: EditorMode) => void;
  onMarkdownViewModeChange: (mode: MarkdownViewMode) => void;
};

export default function ContentEditorField({
  body,
  markdownBody,
  editorMode,
  markdownViewMode,
  onBodyChange,
  onMarkdownBodyChange,
  onEditorModeChange,
  onMarkdownViewModeChange,
}: ContentEditorFieldProps) {
  const markdownTextareaRef = useRef<HTMLTextAreaElement | null>(null);

  const handleEditorModeChange = (next: EditorMode) => {
    if (editorMode === "markdown" && next === "rich") {
      onBodyChange(renderMarkdownToHtml(markdownBody));
    }
    if (editorMode === "rich" && next === "markdown") {
      onMarkdownBodyChange(renderHtmlToMarkdown(body));
    }
    onEditorModeChange(next);
    onMarkdownViewModeChange("write");
  };

  const handleMarkdownImagePaste = async (event: ClipboardEvent<HTMLTextAreaElement>) => {
    const files = Array.from(event.clipboardData?.items || [])
      .map((item) => (item.kind === "file" ? item.getAsFile() : null))
      .filter((file): file is File => Boolean(file) && file!.type.startsWith("image/"));
    if (files.length === 0) return;

    event.preventDefault();
    const textarea = markdownTextareaRef.current;
    const start = textarea?.selectionStart ?? markdownBody.length;
    const end = textarea?.selectionEnd ?? markdownBody.length;

    try {
      const urls = await Promise.all(files.map((file) => uploadImageAndGetURL(file)));
      const insertion = urls
        .map((url, index) => {
          const file = files[index];
          const alt = (file?.name || "image").replace(/\.[^/.]+$/, "");
          return `![${alt}](${url})`;
        })
        .join("\n");
      const next = `${markdownBody.slice(0, start)}${insertion}${markdownBody.slice(end)}`;
      onMarkdownBodyChange(next);
    } catch (err) {
      console.error(err);
      alert("Failed to upload image.");
    }
  };

  return (
    <div className="admin-field">
      <div className="admin-field-head">
        <span>Content</span>
        <label className="admin-editor-mode">
          <span>Editor mode</span>
          <select
            value={editorMode}
            onChange={(e) => handleEditorModeChange(e.target.value as EditorMode)}
          >
            <option value="rich">Rich editor</option>
            <option value="markdown">Markdown</option>
          </select>
        </label>
      </div>
      {editorMode === "rich" ? (
        <RichEditor value={body} onChange={onBodyChange} />
      ) : (
        <div className="admin-markdown-panel">
          <div className="admin-markdown-tabs">
            <button
              type="button"
              className={markdownViewMode === "write" ? "is-active" : ""}
              onClick={() => onMarkdownViewModeChange("write")}
            >
              Write
            </button>
            <button
              type="button"
              className={markdownViewMode === "preview" ? "is-active" : ""}
              onClick={() => onMarkdownViewModeChange("preview")}
            >
              Preview
            </button>
          </div>
          {markdownViewMode === "write" ? (
            <textarea
              ref={markdownTextareaRef}
              value={markdownBody}
              rows={14}
              onPaste={(e) => void handleMarkdownImagePaste(e)}
              onChange={(e) => onMarkdownBodyChange(e.target.value)}
              placeholder="Write Markdown here..."
            />
          ) : (
            <div
              className="admin-markdown-preview"
              dangerouslySetInnerHTML={{ __html: renderMarkdownToHtml(markdownBody) }}
            />
          )}
        </div>
      )}
    </div>
  );
}
