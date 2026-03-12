import { useEffect, useMemo, useState, type KeyboardEvent } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { ClientResponseError } from "pocketbase";
import { pb } from "../lib/pb";
import { buildExcerpt, normalizeMarkdownLinksInHtml, parseTags } from "../utils/text";
import { looksLikeHtml, normalizeFencedCodeBlocksInHtml, renderMarkdownToHtml } from "../utils/markdown";
import SaveButton from "./components/SaveButton";
import ContentEditorField, { type EditorMode, type MarkdownViewMode } from "./components/ContentEditorField";
import FormStatusMessage from "./components/FormStatusMessage";
import PublishFields from "./components/PublishFields";
import TitleSlugFields from "./components/TitleSlugFields";
import TranslationStatusModal from "./components/TranslationStatusModal";
import useUnsavedChangesGuard from "./hooks/useUnsavedChangesGuard";
import useEditorFormState from "./hooks/useEditorFormState";
import usePublishState from "./hooks/usePublishState";
import useTitleSlugState from "./hooks/useTitleSlugState";
import { validateBody, validateSlug, validateTitle } from "./validation";
import type { TranslationJobRecord } from "../lib/pb";

type EditorPostRecord = {
  id: string;
  title?: string;
  slug?: string;
  body?: string;
  content?: string;
  excerpt?: string;
  tags?: string;
  category?: string;
  author?: string;
  published_at?: string;
  published?: boolean;
};

type EditorPostTranslationRecord = EditorPostRecord & {
  locale?: string;
  source_post?: string;
  translation_done?: boolean;
};

type FieldErrors = {
  title?: string;
  slug?: string;
  body?: string;
  tags?: string;
};

const findDuplicateTags = (values: string[]) => {
  const seen = new Map<string, string>();
  const duplicates = new Set<string>();
  values.forEach((tag) => {
    const normalized = tag.trim().toLowerCase();
    if (!normalized) return;
    const first = seen.get(normalized);
    if (first) {
      duplicates.add(first);
      return;
    }
    seen.set(normalized, tag);
  });
  return Array.from(duplicates);
};

const normalizeLocale = (value: string) =>
  value
    .trim()
    .toLowerCase()
    .replace(/_/g, "-");

const parseLocaleList = (value?: string) =>
  (value || "")
    .split(/[\s,;]+/)
    .map((item) => normalizeLocale(item))
    .filter(Boolean);

const escapeFilter = (value: string) => value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');

export default function AdminPostEditor() {
  const { id } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [body, setBody] = useState("");
  const [markdownBody, setMarkdownBody] = useState("");
  const [editorMode, setEditorMode] = useState<EditorMode>("rich");
  const [markdownViewMode, setMarkdownViewMode] = useState<MarkdownViewMode>("write");
  const [excerpt, setExcerpt] = useState("");
  const [tags, setTags] = useState("");
  const [tagInput, setTagInput] = useState("");
  const [category, setCategory] = useState("");
  const [author, setAuthor] = useState("");
  const [publishedAt, setPublishedAt] = useState("");
  const [published, setPublished] = useState(true);
  const [featuredImage, setFeaturedImage] = useState<File | null>(null);
  const [attachments, setAttachments] = useState<File[]>([]);
  const [authors, setAuthors] = useState<Array<{ id: string; name?: string; email?: string }>>([]);
  const [categories, setCategories] = useState<string[]>([]);
  const [tagOptions, setTagOptions] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [slugEditedManually, setSlugEditedManually] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [activeTagSuggestion, setActiveTagSuggestion] = useState(-1);
  const [activeCategorySuggestion, setActiveCategorySuggestion] = useState(-1);
  const [excerptLength, setExcerptLength] = useState(0);
  const { saveMessage, clearSaveMessage, isDirty, markDirty, markSaved } = useEditorFormState();

  const [sourcePostId, setSourcePostId] = useState("");
  const [sourceLocale, setSourceLocale] = useState("ja");
  const [selectedLocale, setSelectedLocale] = useState("ja");
  const [localeOptions, setLocaleOptions] = useState<string[]>(["ja"]);
  const [sourceRecord, setSourceRecord] = useState<EditorPostRecord | null>(null);
  const [localeRecords, setLocaleRecords] = useState<Record<string, EditorPostTranslationRecord>>({});
  const [translationEnabled, setTranslationEnabled] = useState(false);
  const [translationModalOpen, setTranslationModalOpen] = useState(false);
  const [translationJob, setTranslationJob] = useState<TranslationJobRecord | null>(null);
  const [translationJobLoading, setTranslationJobLoading] = useState(false);

  const currentTags = useMemo(() => parseTags(tags), [tags]);
  const tagSuggestions = useMemo(
    () =>
      tagOptions
        .filter((tag) => !currentTags.some((item) => item.toLowerCase() === tag.toLowerCase()))
        .filter((tag) => (tagInput.trim() ? tag.toLowerCase().includes(tagInput.trim().toLowerCase()) : true))
        .slice(0, 20),
    [currentTags, tagInput, tagOptions]
  );
  const categorySuggestions = useMemo(
    () =>
      categories
        .filter((item) => item.toLowerCase() !== category.trim().toLowerCase())
        .filter((item) => (category.trim() ? item.toLowerCase().includes(category.trim().toLowerCase()) : true))
        .slice(0, 20),
    [categories, category]
  );

  const setFieldError = (field: keyof FieldErrors, message?: string) => {
    setFieldErrors((prev) => {
      const next = { ...prev };
      if (message) next[field] = message;
      else delete next[field];
      return next;
    });
  };

  const validateTags = (value: string) => {
    const duplicates = findDuplicateTags(parseTags(value));
    if (duplicates.length > 0) return `Duplicate tags: ${duplicates.join(", ")}`;
    return undefined;
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

  const applyRecordToForm = (record: EditorPostRecord) => {
    const loadedBody = record.body || "";
    const markdownMode = !looksLikeHtml(loadedBody) && loadedBody.trim() !== "";
    setError("");
    setTitle(record.title || "");
    setSlug(record.slug || "");
    setBody(loadedBody);
    setMarkdownBody(markdownMode ? loadedBody : "");
    setEditorMode(markdownMode ? "markdown" : "rich");
    setMarkdownViewMode("write");
    setExcerpt(record.excerpt || "");
    setTags(record.tags || "");
    setTagInput("");
    setCategory(record.category || "");
    setAuthor(record.author || "");
    setPublishedAt(record.published_at ? record.published_at.slice(0, 16) : "");
    setPublished(Boolean(record.published));
    setFeaturedImage(null);
    setAttachments([]);
    setFieldErrors({});
    setActiveTagSuggestion(-1);
    setActiveCategorySuggestion(-1);
    markSaved();
  };

  const applyDraftFromSource = (locale: string, source: EditorPostRecord, sourceId: string) => {
    applyRecordToForm({
      title: source.title,
      slug: source.slug,
      body: source.body,
      excerpt: source.excerpt,
      tags: source.tags,
      category: source.category,
      author: source.author,
      published_at: source.published_at,
      published: source.published,
      locale,
      source_post: sourceId,
    });
  };

  useEffect(() => {
    let alive = true;

    const loadLocaleConfig = async () => {
      try {
        const settingsRes = await pb.collection("settings").getList(1, 1, {
          fields: "site_language,translation_source_locale,translation_locales,enable_post_translation",
        });
        const settings = settingsRes.items[0] || {};
        const src = normalizeLocale(String(settings.translation_source_locale || settings.site_language || "ja")) || "ja";
        const targets = parseLocaleList(String(settings.translation_locales || ""));
        const enabled = Boolean(settings.enable_post_translation) && targets.length > 0;
        return { src, targets, enabled };
      } catch {
        return { src: "ja", targets: ["en"], enabled: false };
      }
    };

    const loadPost = async () => {
      const localeConfig = await loadLocaleConfig();
      if (!alive) return;

      if (!id || id === "new") {
        setTitle("");
        setSlug("");
        setBody("");
        setMarkdownBody("");
        setEditorMode("rich");
        setMarkdownViewMode("write");
        setExcerpt("");
        setTags("");
        setTagInput("");
        setCategory("");
        setAuthor("");
        setPublishedAt("");
        setPublished(true);
        setFeaturedImage(null);
        setAttachments([]);
        setSourcePostId("");
        setSourceLocale(localeConfig.src);
        setSelectedLocale(localeConfig.src);
        setLocaleOptions(Array.from(new Set([localeConfig.src, ...localeConfig.targets])));
        setTranslationEnabled(localeConfig.enabled);
        setTranslationJob(null);
        setTranslationModalOpen(false);
        setLocaleRecords({});
        setSourceRecord(null);
        setSlugEditedManually(false);
        setFieldErrors({});
        setActiveTagSuggestion(-1);
        setActiveCategorySuggestion(-1);
        markSaved();
        return;
      }

      try {
        const source = (await pb.collection("posts").getOne(id)) as unknown as EditorPostRecord;
        const parentId = source.id;
        const inferredSourceLocale = normalizeLocale(localeConfig.src || "ja") || "ja";
        const translations = (await pb.collection("post_translations").getFullList({
          filter: `source_post = "${escapeFilter(parentId)}"`,
          sort: "locale",
        })) as unknown as EditorPostTranslationRecord[];

        const byLocale: Record<string, EditorPostTranslationRecord> = {};
        translations.forEach((item) => {
          const loc = normalizeLocale(item.locale || "");
          if (loc) byLocale[loc] = item;
        });

        const options = Array.from(
          new Set([inferredSourceLocale, ...localeConfig.targets.map(normalizeLocale), ...Object.keys(byLocale)]),
        ).filter(Boolean);

        const initialLocale = inferredSourceLocale;

        if (!alive) return;
        setSourcePostId(parentId);
        setSourceLocale(inferredSourceLocale);
        setSourceRecord(source);
        setLocaleRecords(byLocale);
        setLocaleOptions(options);
        setTranslationEnabled(localeConfig.enabled);
        setSelectedLocale(initialLocale);
        setSlugEditedManually(true);

        applyRecordToForm(source);
      } catch (err) {
        if (!alive) return;
        setError("Failed to load post. Check permissions or post ID.");
        console.error(err);
      }
    };

    loadPost();

    return () => {
      alive = false;
    };
  }, [id]);

  useEffect(() => {
    if (activeTagSuggestion < tagSuggestions.length) return;
    setActiveTagSuggestion(-1);
  }, [activeTagSuggestion, tagSuggestions.length]);

  useEffect(() => {
    if (activeCategorySuggestion < categorySuggestions.length) return;
    setActiveCategorySuggestion(-1);
  }, [activeCategorySuggestion, categorySuggestions.length]);

  useUnsavedChangesGuard(isDirty);

  useEffect(() => {
    const state = location.state as { saved?: boolean; created?: boolean; translationQueued?: boolean } | null;
    if (!state?.saved) return;
    setError("");
    markSaved(state.created ? "Post saved (new post created)." : "Post saved.");
    if (state.translationQueued && id && id !== "new") {
      setTranslationModalOpen(true);
    }
    navigate(location.pathname, { replace: true });
  }, [id, location.pathname, location.state, markSaved, navigate]);

  useEffect(() => {
    if (!translationModalOpen) return;
    const currentSourceId = sourcePostId || (id && id !== "new" ? id : "");
    if (!currentSourceId) return;

    let active = true;
    let timer: number | undefined;

    const loadJob = async () => {
      setTranslationJobLoading(true);
      try {
        const job = await pb
          .collection("translation_jobs")
          .getFirstListItem<TranslationJobRecord>(`source_post = "${escapeFilter(currentSourceId)}"`);
        if (!active) return;
        setTranslationJob(job);
        if (job.status === "queued" || job.status === "running") {
          timer = window.setTimeout(() => {
            void loadJob();
          }, 1500);
        }
      } catch (err) {
        if (!active) return;
        if (err instanceof ClientResponseError && err.status === 404) {
          timer = window.setTimeout(() => {
            void loadJob();
          }, 1000);
          return;
        }
        console.error(err);
      } finally {
        if (active) setTranslationJobLoading(false);
      }
    };

    void loadJob();
    return () => {
      active = false;
      if (timer) window.clearTimeout(timer);
    };
  }, [id, sourcePostId, translationModalOpen]);

  useEffect(() => {
    pb.collection("cms_users")
      .getFullList({ sort: "name" })
      .then((items) => setAuthors(items as Array<{ id: string; name?: string; email?: string }>))
      .catch(() => setAuthors([]));
  }, []);

  useEffect(() => {
    pb.collection("posts")
      .getFullList({ fields: "category,tags" })
      .then((items: any[]) => {
        const categorySet = new Set<string>();
        const tagSet = new Set<string>();
        items.forEach((item) => {
          if (item.category) categorySet.add(String(item.category));
          parseTags(item.tags).forEach((tag) => tagSet.add(tag));
        });
        setCategories(Array.from(categorySet));
        setTagOptions(Array.from(tagSet));
      })
      .catch(() => {
        setCategories([]);
        setTagOptions([]);
      });
  }, []);

  useEffect(() => {
    pb.collection("settings")
      .getList(1, 1)
      .then((res) => {
        const length = Number(res.items?.[0]?.excerpt_length ?? 0);
        setExcerptLength(Number.isFinite(length) ? length : 0);
      })
      .catch(() => setExcerptLength(0));
  }, []);

  const switchLocale = (nextLocaleRaw: string) => {
    const nextLocale = normalizeLocale(nextLocaleRaw);
    if (!nextLocale || nextLocale === selectedLocale) return;
    if (isDirty && !window.confirm("You have unsaved changes. Switch locale without saving?")) {
      return;
    }
    setSelectedLocale(nextLocale);
    if (nextLocale === sourceLocale) {
      if (sourceRecord) {
        applyRecordToForm(sourceRecord);
      }
      return;
    }
    const existing = localeRecords[nextLocale];
    if (existing) {
      applyRecordToForm(existing);
      return;
    }
    if (sourceRecord) {
      applyDraftFromSource(nextLocale, sourceRecord, sourcePostId);
    }
  };

  const addTag = (tag: string) => {
    const trimmed = tag.trim();
    if (!trimmed) return;
    const merged = [...currentTags, trimmed]
      .filter((item, index, arr) => arr.findIndex((ref) => ref.toLowerCase() === item.toLowerCase()) === index)
      .join(", ");
    setTags(merged);
    setTagInput("");
    markDirty();
    setActiveTagSuggestion(-1);
    setFieldError("tags", validateTags(merged));
  };

  const applyCategoryInput = () => {
    setCategory((prev) => prev.trim());
    setActiveCategorySuggestion(-1);
  };

  const applyTagInputOnBlur = () => {
    if (!tagInput.trim()) return;
    if (activeTagSuggestion >= 0 && tagSuggestions[activeTagSuggestion]) {
      addTag(tagSuggestions[activeTagSuggestion]);
      return;
    }
    addTag(tagInput);
  };

  const removeTag = (tag: string) => {
    const merged = currentTags
      .filter((item) => item.toLowerCase() !== tag.toLowerCase())
      .join(", ");
    setTags(merged);
    markDirty();
    setFieldError("tags", validateTags(merged));
  };

  const onTagInputKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      if (tagSuggestions.length === 0) return;
      setActiveTagSuggestion((prev) => (prev + 1) % tagSuggestions.length);
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      if (tagSuggestions.length === 0) return;
      setActiveTagSuggestion((prev) => (prev <= 0 ? tagSuggestions.length - 1 : prev - 1));
      return;
    }
    if (event.key === "Enter") {
      event.preventDefault();
      if (activeTagSuggestion >= 0 && tagSuggestions[activeTagSuggestion]) {
        addTag(tagSuggestions[activeTagSuggestion]);
        return;
      }
      addTag(tagInput);
    }
  };

  const onCategoryInputKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      if (categorySuggestions.length === 0) return;
      setActiveCategorySuggestion((prev) => (prev + 1) % categorySuggestions.length);
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      if (categorySuggestions.length === 0) return;
      setActiveCategorySuggestion((prev) => (prev <= 0 ? categorySuggestions.length - 1 : prev - 1));
      return;
    }
    if (event.key === "Enter") {
      event.preventDefault();
      const next =
        activeCategorySuggestion >= 0 && categorySuggestions[activeCategorySuggestion]
          ? categorySuggestions[activeCategorySuggestion]
          : category.trim();
      setCategory(next);
      markDirty();
      setActiveCategorySuggestion(-1);
    }
  };

  const save = async () => {
    if (saving) return;
    setError("");
    clearSaveMessage();
    const isMarkdownMode = editorMode === "markdown";
    const sourceBody = isMarkdownMode ? markdownBody : body;
    const normalizedBody = isMarkdownMode
      ? renderMarkdownToHtml(sourceBody, { highlightCode: false })
      : normalizeFencedCodeBlocksInHtml(normalizeMarkdownLinksInHtml(sourceBody), { highlightCode: false });
    const autoExcerpt = excerptLength > 0;
    const finalExcerpt = autoExcerpt ? buildExcerpt(normalizedBody, excerptLength) : excerpt;
    const trimmedTitle = title.trim();
    const trimmedSlug = slug.trim();
    const trimmedBody = sourceBody.trim();
    const nextErrors: FieldErrors = {
      title: validateTitle(trimmedTitle),
      slug: validateSlug(trimmedSlug),
      body: validateBody(trimmedBody),
      tags: validateTags(tags),
    };
    const hasErrors = Object.values(nextErrors).some(Boolean);
    setFieldErrors(nextErrors);
    if (hasErrors) {
      setError("Please fix validation errors.");
      return;
    }

    const form = new FormData();
    form.set("title", trimmedTitle);
    form.set("slug", trimmedSlug);
    form.set("body", normalizedBody);
    form.set("excerpt", (finalExcerpt || buildExcerpt(normalizedBody)).trim());
    if (tags.trim() !== "") form.set("tags", tags.trim());
    if (category.trim() !== "") form.set("category", category.trim());
    if (author.trim() !== "") form.set("author", author.trim());
    form.set("published_at", publishedAt ? new Date(publishedAt).toISOString() : new Date().toISOString());
    form.set("published", String(published));
    if (featuredImage) {
      form.set("featured_image", featuredImage);
    }
    if (attachments.length > 0) {
      attachments.forEach((file) => form.append("attachments", file));
    }

    setSaving(true);
    try {
      const shouldQueueTranslation = selectedLocale === sourceLocale && translationEnabled;
      if (!id || id === "new" || sourcePostId === "") {
        const created = (await pb.collection("posts").create(form)) as unknown as EditorPostRecord;
        markSaved();
        navigate(`/posts/${created.id}`, {
          state: { saved: true, created: true, translationQueued: shouldQueueTranslation },
        });
        return;
      }

      if (selectedLocale === sourceLocale) {
        const updated = (await pb.collection("posts").update(sourcePostId, form)) as unknown as EditorPostRecord;
        setSourceRecord(updated);
        if (shouldQueueTranslation) {
          setTranslationJob(null);
          setTranslationModalOpen(true);
        }
      } else {
        form.set("source_post", sourcePostId);
        form.set("locale", selectedLocale);
        form.set("translation_done", "true");
        const currentTranslation = localeRecords[selectedLocale];
        const saved = currentTranslation
          ? ((await pb.collection("post_translations").update(currentTranslation.id, form)) as unknown as EditorPostTranslationRecord)
          : ((await pb.collection("post_translations").create(form)) as unknown as EditorPostTranslationRecord);

        setLocaleRecords((prev) => ({ ...prev, [selectedLocale]: saved }));
        setLocaleOptions((prev) => (prev.includes(selectedLocale) ? prev : [...prev, selectedLocale]));
      }
      markSaved("Post saved.");
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
        <h1>{id === "new" ? "New Post" : "Edit Post"}</h1>
        <SaveButton onClick={save} saving={saving} />
      </header>
      <FormStatusMessage error={error} success={saveMessage} />
      <TranslationStatusModal
        open={translationModalOpen}
        job={translationJob}
        loading={translationJobLoading}
        onClose={() => setTranslationModalOpen(false)}
      />
      <div className="admin-form">
        {id !== "new" && localeOptions.length > 0 && (
          <div className="admin-field">
            <span>Edit locale</span>
            <div className="admin-tag-suggestions">
              {localeOptions.map((locale) => {
                const selected = locale === selectedLocale;
                return (
                  <button
                    type="button"
                    key={locale}
                    onClick={() => switchLocale(locale)}
                    disabled={saving}
                    style={{ opacity: selected ? 1 : 0.6 }}
                  >
                    {locale === sourceLocale ? `${locale} (source)` : locale}
                  </button>
                );
              })}
            </div>
          </div>
        )}
        <TitleSlugFields
          title={title}
          slug={slug}
          slugEditedManually={slugEditedManually}
          titleError={fieldErrors.title}
          slugError={fieldErrors.slug}
          onTitleChange={onTitleChange}
          onSlugChange={onSlugChange}
          onAutoSlug={onAutoSlug}
        />
        <PublishFields
          publishedAt={publishedAt}
          published={published}
          onPublishedAtChange={onPublishedAtChange}
          onPublishedChange={onPublishedChange}
        />
        <label>
          Category
          <input
            value={category}
            onChange={(e) => {
              setCategory(e.target.value);
              markDirty();
              setActiveCategorySuggestion(-1);
            }}
            onKeyDown={onCategoryInputKeyDown}
            onBlur={applyCategoryInput}
            enterKeyHint="done"
          />
        </label>
        {categorySuggestions.length > 0 && (
          <div className="admin-tag-suggestions">
            {categorySuggestions.map((item) => (
              <button
                type="button"
                key={item}
                className={categorySuggestions[activeCategorySuggestion] === item ? "is-active" : ""}
                onClick={() => {
                  setCategory(item);
                  markDirty();
                  setActiveCategorySuggestion(-1);
                }}
              >
                {item}
              </button>
            ))}
          </div>
        )}
        <label>
          Author
          <select
            value={author}
            onChange={(e) => {
              setAuthor(e.target.value);
              markDirty();
            }}
          >
            <option value="">(none)</option>
            {authors.map((user) => (
              <option key={user.id} value={user.id}>
                {user.name || user.email || user.id}
              </option>
            ))}
          </select>
        </label>
        <label>
          Tags
          <input
            value={tagInput}
            onChange={(e) => {
              const next = e.target.value;
              setTagInput(next);
              markDirty();
              setActiveTagSuggestion(-1);
            }}
            onKeyDown={onTagInputKeyDown}
            onBlur={applyTagInputOnBlur}
            placeholder="Type tag and press Enter"
            enterKeyHint="done"
          />
        </label>
        {fieldErrors.tags && <p className="admin-error-inline">{fieldErrors.tags}</p>}
        {currentTags.length > 0 && (
          <div className="admin-tag-suggestions">
            {currentTags.map((tag) => (
              <button type="button" key={`selected-${tag}`} onClick={() => removeTag(tag)}>
                {tag} ×
              </button>
            ))}
          </div>
        )}
        {tagOptions.length > 0 && (
          <div className="admin-tag-suggestions">
            {tagSuggestions
              .map((tag) => (
                <button
                  type="button"
                  key={tag}
                  className={tagSuggestions[activeTagSuggestion] === tag ? "is-active" : ""}
                  onClick={() => addTag(tag)}
                >
                  {tag}
                </button>
              ))}
          </div>
        )}
        <label>
          Featured image
          <input
            type="file"
            accept="image/*"
            onChange={(e) => {
              setFeaturedImage(e.target.files ? e.target.files[0] : null);
              markDirty();
            }}
          />
        </label>
        <label>
          Attachments
          <input
            type="file"
            multiple
            onChange={(e) => {
              setAttachments(e.target.files ? Array.from(e.target.files) : []);
              markDirty();
            }}
          />
        </label>
        <label>
          Excerpt
          <textarea
            value={
              excerptLength > 0
                ? buildExcerpt(
                    editorMode === "markdown" ? renderMarkdownToHtml(markdownBody, { highlightCode: false }) : body,
                    excerptLength
                  )
                : excerpt
            }
            onChange={(e) => {
              setExcerpt(e.target.value);
              markDirty();
            }}
            rows={3}
            disabled={excerptLength > 0}
          />
        </label>
        {excerptLength > 0 && (
          <p className="admin-note">
            Excerpt is auto-generated from content ({excerptLength} chars).
          </p>
        )}
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
