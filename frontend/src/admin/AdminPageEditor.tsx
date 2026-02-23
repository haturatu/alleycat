import { useEffect, useRef, useState, type ClipboardEvent } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ClientResponseError } from "pocketbase";
import { pb } from "../lib/pb";
import { normalizeMarkdownLinksInHtml, slugify, stripHtml } from "../utils/text";
import { looksLikeHtml, renderMarkdownToHtml } from "../utils/markdown";
import RichEditor from "./RichEditor";
import { uploadImageAndGetURL } from "./mediaUpload";
import SaveButton from "./components/SaveButton";
import useUnsavedChangesGuard from "./hooks/useUnsavedChangesGuard";
import useEditorFormState from "./hooks/useEditorFormState";

type FieldErrors = {
  title?: string;
  slug?: string;
  url?: string;
  body?: string;
};

type EditorMode = "rich" | "markdown";
type MarkdownViewMode = "write" | "preview";

export default function AdminPageEditor() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [url, setUrl] = useState("");
  const [menuVisible, setMenuVisible] = useState(false);
  const [menuOrder, setMenuOrder] = useState(0);
  const [menuTitle, setMenuTitle] = useState("");
  const [body, setBody] = useState("");
  const [markdownBody, setMarkdownBody] = useState("");
  const [editorMode, setEditorMode] = useState<EditorMode>("rich");
  const [markdownViewMode, setMarkdownViewMode] = useState<MarkdownViewMode>("write");
  const [publishedAt, setPublishedAt] = useState("");
  const [published, setPublished] = useState(true);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [slugEditedManually, setSlugEditedManually] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const { saveMessage, clearSaveMessage, isDirty, markDirty, markSaved } = useEditorFormState();
  const markdownTextareaRef = useRef<HTMLTextAreaElement | null>(null);

  const setFieldError = (field: keyof FieldErrors, message?: string) => {
    setFieldErrors((prev) => {
      const next = { ...prev };
      if (message) next[field] = message;
      else delete next[field];
      return next;
    });
  };

  const validateTitle = (value: string) => (value.trim() ? undefined : "Title is required.");
  const validateSlug = (value: string) => {
    const trimmed = value.trim();
    if (!trimmed) return "Slug is required.";
    if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(trimmed)) {
      return "Use lowercase letters, numbers, and hyphens.";
    }
    return undefined;
  };
  const validateURL = (value: string) => {
    const trimmed = value.trim();
    if (!trimmed) return undefined;
    if (!trimmed.startsWith("/")) return "URL must start with '/'.";
    return undefined;
  };
  const validateBody = (value: string) => (value.trim() ? undefined : "Content is required.");

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
      setMarkdownBody(next);
      markDirty();
      setFieldError("body", validateBody(next));
    } catch (err) {
      console.error(err);
      alert("Failed to upload image.");
    }
  };

  useEffect(() => {
    if (!id || id === "new") {
      setSlugEditedManually(false);
      setFieldErrors({});
      setBody("");
      setMarkdownBody("");
      setEditorMode("rich");
      setMarkdownViewMode("write");
      markSaved();
      return;
    }
    pb.collection("pages")
      .getOne(id)
      .then((record) => {
        setError("");
        setTitle(record.title || "");
        setSlug(record.slug || "");
        setUrl(record.url || "");
        setMenuVisible(Boolean(record.menuVisible));
        setMenuOrder(record.menuOrder || 0);
        setMenuTitle(record.menuTitle || "");
        const loadedBody = String(record.body || "");
        const markdownMode = !looksLikeHtml(loadedBody) && loadedBody.trim() !== "";
        setBody(loadedBody);
        setMarkdownBody(markdownMode ? loadedBody : "");
        setEditorMode(markdownMode ? "markdown" : "rich");
        setMarkdownViewMode("write");
        setPublishedAt(record.published_at ? record.published_at.slice(0, 16) : "");
        setPublished(Boolean(record.published));
        setSlugEditedManually(true);
        setFieldErrors({});
        markSaved();
      })
      .catch((err) => {
        setError("Failed to load page. Check permissions or page ID.");
        console.error(err);
      });
  }, [id, navigate]);

  useUnsavedChangesGuard(isDirty);

  const save = async () => {
    if (saving) return;
    setError("");
    clearSaveMessage();

    const trimmedTitle = title.trim();
    const trimmedSlug = slug.trim();
    const isMarkdownMode = editorMode === "markdown";
    const sourceBody = isMarkdownMode ? markdownBody : body;
    const normalizedBody =
      isMarkdownMode
        ? renderMarkdownToHtml(sourceBody)
        : normalizeMarkdownLinksInHtml(sourceBody);
    const trimmedBody = sourceBody.trim();
    const resolvedUrl = url || `/${slug}/`;
    const nextErrors: FieldErrors = {
      title: validateTitle(trimmedTitle),
      slug: validateSlug(trimmedSlug),
      url: validateURL(resolvedUrl),
      body: validateBody(trimmedBody),
    };
    const hasErrors = Object.values(nextErrors).some(Boolean);
    setFieldErrors(nextErrors);
    if (hasErrors) {
      setError("Please fix validation errors.");
      return;
    }

    const payload = {
      title: trimmedTitle,
      slug: trimmedSlug,
      url: resolvedUrl,
      menuVisible,
      menuOrder,
      menuTitle,
      body: normalizedBody,
      published_at: publishedAt ? new Date(publishedAt).toISOString() : new Date().toISOString(),
      published,
    };

    setSaving(true);
    try {
      if (!id || id === "new") {
        await pb.collection("pages").create(payload);
      } else {
        await pb.collection("pages").update(id, payload);
      }
      markSaved("Page saved.");
      navigate("/pages");
    } catch (err) {
      if (err instanceof ClientResponseError) {
        const details = err.response?.data as Record<string, { message?: string }> | undefined;
        const detailText = details
          ? Object.entries(details)
              .map(([field, value]) => `${field}: ${value?.message || "invalid"}`)
              .join(", ")
          : "";
        setError(detailText ? `Save failed: ${detailText}` : "Save failed.");
      } else {
        setError("Save failed.");
      }
      console.error(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <section>
      <header className="admin-header">
        <h1>{id === "new" ? "New Page" : "Edit Page"}</h1>
        <SaveButton onClick={save} saving={saving} />
      </header>
      {error && <p className="admin-error">{error}</p>}
      {saveMessage && <p className="admin-success">{saveMessage}</p>}
      <div className="admin-form">
        <label>
          Title
          <input
            value={title}
            onChange={(e) => {
              const next = e.target.value;
              setTitle(next);
              markDirty();
              setFieldError("title", validateTitle(next));
              if (!slugEditedManually) {
                const nextSlug = slugify(next);
                setSlug(nextSlug);
                setFieldError("slug", validateSlug(nextSlug));
              }
            }}
          />
        </label>
        {fieldErrors.title && <p className="admin-error-inline">{fieldErrors.title}</p>}
        <label>
          Slug
          <div className="admin-inline">
            <input
              value={slug}
              onChange={(e) => {
                const next = e.target.value;
                setSlug(next);
                setSlugEditedManually(true);
                markDirty();
                setFieldError("slug", validateSlug(next));
              }}
            />
            <button
              type="button"
              disabled={saving}
              onClick={() => {
                const auto = slugify(title);
                setSlug(auto);
                setSlugEditedManually(false);
                markDirty();
                setFieldError("slug", validateSlug(auto));
              }}
            >
              Auto
            </button>
          </div>
        </label>
        {fieldErrors.slug ? (
          <p className="admin-error-inline">{fieldErrors.slug}</p>
        ) : (
          <p className="admin-note">
            {slugEditedManually ? "Slug is locked (manual)." : "Slug follows title automatically."}
          </p>
        )}
        <label>
          URL
          <input
            value={url}
            onChange={(e) => {
              const next = e.target.value;
              setUrl(next);
              markDirty();
              const previewURL = next.trim() || `/${slugify(title)}/`;
              setFieldError("url", validateURL(previewURL));
            }}
            placeholder="/ab/"
          />
        </label>
        {fieldErrors.url && <p className="admin-error-inline">{fieldErrors.url}</p>}
        <label className="admin-check admin-check-right">
          <span>Show in menu</span>
          <input
            type="checkbox"
            checked={menuVisible}
            onChange={(e) => {
              setMenuVisible(e.target.checked);
              markDirty();
            }}
          />
        </label>
        <label>
          Published at
          <input
            type="datetime-local"
            value={publishedAt}
            onChange={(e) => {
              setPublishedAt(e.target.value);
              markDirty();
            }}
          />
        </label>
        <label>
          Menu order
          <input
            type="number"
            value={menuOrder}
            onChange={(e) => {
              setMenuOrder(Number(e.target.value));
              markDirty();
            }}
          />
        </label>
        <label>
          Menu title
          <input
            value={menuTitle}
            onChange={(e) => {
              setMenuTitle(e.target.value);
              markDirty();
            }}
          />
        </label>
        <label className="admin-check admin-check-right">
          <span>Published</span>
          <input
            type="checkbox"
            checked={published}
            onChange={(e) => {
              setPublished(e.target.checked);
              markDirty();
            }}
          />
        </label>
        <div className="admin-field">
          <div className="admin-field-head">
            <span>Content</span>
            <label className="admin-editor-mode">
              <span>Editor mode</span>
              <select
                value={editorMode}
                onChange={(e) => {
                  const next = e.target.value as EditorMode;
                  if (editorMode === "markdown" && next === "rich") {
                    setBody(renderMarkdownToHtml(markdownBody));
                  }
                  setEditorMode(next);
                  if (next === "markdown" && markdownBody.trim() === "") {
                    setMarkdownBody(looksLikeHtml(body) ? stripHtml(body) : body);
                  }
                  setMarkdownViewMode("write");
                  markDirty();
                }}
              >
                <option value="rich">Rich editor (HTML)</option>
                <option value="markdown">Markdown (RFC 7763)</option>
              </select>
            </label>
          </div>
          {editorMode === "rich" ? (
            <RichEditor
              value={body}
              onChange={(value) => {
                setBody(value);
                markDirty();
                setFieldError("body", validateBody(value));
              }}
            />
          ) : (
            <div className="admin-markdown-panel">
              <div className="admin-markdown-tabs">
                <button
                  type="button"
                  className={markdownViewMode === "write" ? "is-active" : ""}
                  onClick={() => setMarkdownViewMode("write")}
                >
                  Write
                </button>
                <button
                  type="button"
                  className={markdownViewMode === "preview" ? "is-active" : ""}
                  onClick={() => setMarkdownViewMode("preview")}
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
                  onChange={(e) => {
                    setMarkdownBody(e.target.value);
                    markDirty();
                    setFieldError("body", validateBody(e.target.value));
                  }}
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
        {fieldErrors.body && <p className="admin-error-inline">{fieldErrors.body}</p>}
      </div>
    </section>
  );
}
