import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ClientResponseError } from "pocketbase";
import { pb } from "../lib/pb";
import { normalizeMarkdownLinksInHtml, slugify } from "../utils/text";
import { looksLikeHtml, normalizeFencedCodeBlocksInHtml, renderMarkdownToHtml } from "../utils/markdown";
import SaveButton from "./components/SaveButton";
import { AdminCheckboxField, AdminTextField } from "./components/AriaControls";
import ContentEditorField, { type EditorMode, type MarkdownViewMode } from "./components/ContentEditorField";
import FormStatusMessage from "./components/FormStatusMessage";
import PublishFields from "./components/PublishFields";
import TitleSlugFields from "./components/TitleSlugFields";
import useUnsavedChangesGuard from "./hooks/useUnsavedChangesGuard";
import useEditorFormState from "./hooks/useEditorFormState";
import usePublishState from "./hooks/usePublishState";
import useTitleSlugState from "./hooks/useTitleSlugState";
import { validateBody, validateSlug, validateTitle, validateURL } from "./validation";

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

  const { onTitleChange, onSlugChange, onAutoSlug } = useTitleSlugState({
    title,
    slugEditedManually,
    setTitle,
    setSlug,
    setSlugEditedManually,
    markDirty,
    setFieldError,
  });

  const { onPublishedAtChange, onPublishedChange } = usePublishState({
    setPublishedAt,
    setPublished,
    markDirty,
  });

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
        ? renderMarkdownToHtml(sourceBody, { highlightCode: false })
        : normalizeFencedCodeBlocksInHtml(normalizeMarkdownLinksInHtml(sourceBody), { highlightCode: false });
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
      <FormStatusMessage error={error} success={saveMessage} />
      <div className="admin-form">
        <TitleSlugFields
          title={title}
          slug={slug}
          slugEditedManually={slugEditedManually}
          titleError={fieldErrors.title}
          slugError={fieldErrors.slug}
          autoDisabled={saving}
          onTitleChange={onTitleChange}
          onSlugChange={onSlugChange}
          onAutoSlug={onAutoSlug}
        />
        <AdminTextField
          label="URL"
          value={url}
          onChange={(next) => {
            setUrl(next);
            markDirty();
            const previewURL = next.trim() || `/${slugify(title)}/`;
            setFieldError("url", validateURL(previewURL));
          }}
          placeholder="/ab/"
        />
        {fieldErrors.url && <p className="admin-error-inline">{fieldErrors.url}</p>}
        <AdminCheckboxField
          label="Show in menu"
          checked={menuVisible}
          onChange={(checked) => {
            setMenuVisible(checked);
            markDirty();
          }}
        />
        <PublishFields
          publishedAt={publishedAt}
          published={published}
          onPublishedAtChange={onPublishedAtChange}
          onPublishedChange={onPublishedChange}
        />
        <AdminTextField
          label="Menu order"
          type="number"
          value={String(menuOrder)}
          onChange={(value) => {
            setMenuOrder(Number(value));
            markDirty();
          }}
        />
        <AdminTextField
          label="Menu title"
          value={menuTitle}
          onChange={(value) => {
            setMenuTitle(value);
            markDirty();
          }}
        />
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
