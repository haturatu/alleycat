import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ClientResponseError } from "pocketbase";
import { pb } from "@cms/lib/pb";
import { normalizeMarkdownLinksInHtml, slugify } from "@cms/utils/text";
import { looksLikeHtml, normalizeFencedCodeBlocksInHtml, renderMarkdownToHtml } from "@cms/utils/markdown";
import SaveButton from "@cms/ui/SaveButton";
import { AdminCheckboxField, AdminTextField } from "@cms/ui/AriaControls";
import ContentEditorField, { type EditorMode, type MarkdownViewMode } from "@cms/features/editor/components/ContentEditorField";
import FormStatusMessage from "@cms/ui/FormStatusMessage";
import PublishFields from "@cms/features/editor/components/PublishFields";
import TitleSlugFields from "@cms/features/editor/components/TitleSlugFields";
import useAdminPageTitle from "@cms/useAdminPageTitle";
import useUnsavedChangesGuard from "@cms/features/editor/hooks/useUnsavedChangesGuard";
import useEditorFormState from "@cms/features/editor/hooks/useEditorFormState";
import usePublishState from "@cms/features/editor/hooks/usePublishState";
import useTitleSlugState from "@cms/features/editor/hooks/useTitleSlugState";
import { validateBody, validateSlug, validateTitle, validateURL } from "@cms/features/editor/validation";

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
  const { saveMessage, clearSaveMessage, isDirty, lastSavedAt, markDirty, markSaved } = useEditorFormState();
  const titleInputRef = useRef<HTMLInputElement>(null);
  const slugInputRef = useRef<HTMLInputElement>(null);
  const urlInputRef = useRef<HTMLInputElement>(null);

  useAdminPageTitle(id === "new" ? "New Page" : "Edit Page");

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

  const focusFirstError = (errors: FieldErrors) => {
    window.requestAnimationFrame(() => {
      if (errors.title) {
        titleInputRef.current?.focus();
        return;
      }
      if (errors.slug) {
        slugInputRef.current?.focus();
        return;
      }
      if (errors.url) {
        urlInputRef.current?.focus();
        return;
      }
      if (errors.body) {
        const target = document.querySelector(".admin-markdown-panel textarea, .editor .ProseMirror") as HTMLElement | null;
        target?.focus();
      }
    });
  };

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
        : normalizeFencedCodeBlocksInHtml(normalizeMarkdownLinksInHtml(sourceBody));
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
      focusFirstError(nextErrors);
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
      <header className="admin-header admin-editor-header">
        <div>
          <p className="admin-eyebrow">Site Structure</p>
          <h1>{id === "new" ? "New Page" : "Edit Page"}</h1>
        </div>
        <div className="admin-header-actions">
          {saving || isDirty || saveMessage ? (
            <span className="admin-inline-status">
              {saving ? "Saving…" : isDirty ? "Unsaved" : lastSavedAt ? `Saved ${lastSavedAt}` : "Saved"}
            </span>
          ) : null}
          <SaveButton onClick={save} saving={saving} />
        </div>
      </header>
      <FormStatusMessage error={error} success={saveMessage} />
      <div className="admin-editor-shell">
        <div className="admin-editor-main">
          <div className="admin-form-section admin-editor-canvas">
            <p className="admin-section-label">Page content</p>
            <TitleSlugFields
              editorial
              title={title}
              slug={slug}
              slugEditedManually={slugEditedManually}
              titleError={fieldErrors.title}
              slugError={fieldErrors.slug}
              autoDisabled={saving}
              titleInputRef={titleInputRef}
              slugInputRef={slugInputRef}
              onTitleChange={onTitleChange}
              onSlugChange={onSlugChange}
              onAutoSlug={onAutoSlug}
            />
            <AdminTextField
              inputRef={urlInputRef}
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
        </div>
        <aside className="admin-editor-rail">
          <div className="admin-form admin-form-section admin-rail-panel">
            <p className="admin-section-label">Navigation</p>
            <AdminCheckboxField
              label="Show in menu"
              checked={menuVisible}
              onChange={(checked) => {
                setMenuVisible(checked);
                markDirty();
              }}
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
          </div>
          <div className="admin-form admin-form-section admin-rail-panel">
            <p className="admin-section-label">Publishing</p>
            <PublishFields
              publishedAt={publishedAt}
              published={published}
              onPublishedAtChange={onPublishedAtChange}
              onPublishedChange={onPublishedChange}
            />
          </div>
        </aside>
      </div>
    </section>
  );
}
