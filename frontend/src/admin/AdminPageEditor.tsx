import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ClientResponseError } from "pocketbase";
import { pb } from "../lib/pb";
import { normalizeMarkdownLinksInHtml, slugify } from "../utils/text";
import { looksLikeHtml, renderMarkdownToHtml } from "../utils/markdown";
import SaveButton from "./components/SaveButton";
import ContentEditorField, { type EditorMode, type MarkdownViewMode } from "./components/ContentEditorField";
import PublishFields from "./components/PublishFields";
import TitleSlugFields from "./components/TitleSlugFields";
import useUnsavedChangesGuard from "./hooks/useUnsavedChangesGuard";
import useEditorFormState from "./hooks/useEditorFormState";

type FieldErrors = {
  title?: string;
  slug?: string;
  url?: string;
  body?: string;
};

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
        <TitleSlugFields
          title={title}
          slug={slug}
          slugEditedManually={slugEditedManually}
          titleError={fieldErrors.title}
          slugError={fieldErrors.slug}
          autoDisabled={saving}
          onTitleChange={(next) => {
            setTitle(next);
            markDirty();
            setFieldError("title", validateTitle(next));
            if (!slugEditedManually) {
              const nextSlug = slugify(next);
              setSlug(nextSlug);
              setFieldError("slug", validateSlug(nextSlug));
            }
          }}
          onSlugChange={(next) => {
            setSlug(next);
            setSlugEditedManually(true);
            markDirty();
            setFieldError("slug", validateSlug(next));
          }}
          onAutoSlug={() => {
            const auto = slugify(title);
            setSlug(auto);
            setSlugEditedManually(false);
            markDirty();
            setFieldError("slug", validateSlug(auto));
          }}
        />
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
        <PublishFields
          publishedAt={publishedAt}
          published={published}
          onPublishedAtChange={(value) => {
            setPublishedAt(value);
            markDirty();
          }}
          onPublishedChange={(checked) => {
            setPublished(checked);
            markDirty();
          }}
        />
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
        <ContentEditorField
          body={body}
          markdownBody={markdownBody}
          editorMode={editorMode}
          markdownViewMode={markdownViewMode}
          onBodyChange={(value) => {
            setBody(value);
            markDirty();
            setFieldError("body", validateBody(value));
          }}
          onMarkdownBodyChange={(value) => {
            setMarkdownBody(value);
            markDirty();
            setFieldError("body", validateBody(value));
          }}
          onEditorModeChange={(mode) => {
            setEditorMode(mode);
            markDirty();
          }}
          onMarkdownViewModeChange={setMarkdownViewMode}
        />
        {fieldErrors.body && <p className="admin-error-inline">{fieldErrors.body}</p>}
      </div>
    </section>
  );
}
