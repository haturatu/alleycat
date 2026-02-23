import { useEffect, useMemo, useRef, useState, type ClipboardEvent, type KeyboardEvent } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { ClientResponseError } from "pocketbase";
import { pb } from "../lib/pb";
import { buildExcerpt, normalizeMarkdownLinksInHtml, parseTags, slugify, stripHtml } from "../utils/text";
import { looksLikeHtml, renderMarkdownToHtml } from "../utils/markdown";
import RichEditor from "./RichEditor";
import { uploadImageAndGetURL } from "./mediaUpload";
import SaveButton from "./components/SaveButton";
import useUnsavedChangesGuard from "./hooks/useUnsavedChangesGuard";
import useEditorFormState from "./hooks/useEditorFormState";

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

type EditorMode = "rich" | "markdown";
type MarkdownViewMode = "write" | "preview";

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
  const markdownTextareaRef = useRef<HTMLTextAreaElement | null>(null);

  const [sourcePostId, setSourcePostId] = useState("");
  const [sourceLocale, setSourceLocale] = useState("ja");
  const [selectedLocale, setSelectedLocale] = useState("ja");
  const [localeOptions, setLocaleOptions] = useState<string[]>(["ja"]);
  const [sourceRecord, setSourceRecord] = useState<EditorPostRecord | null>(null);
  const [localeRecords, setLocaleRecords] = useState<Record<string, EditorPostTranslationRecord>>({});

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

  const validateTitle = (value: string) => (value.trim() ? undefined : "Title is required.");
  const validateSlug = (value: string) => {
    const trimmed = value.trim();
    if (!trimmed) return "Slug is required.";
    if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(trimmed)) {
      return "Use lowercase letters, numbers, and hyphens.";
    }
    return undefined;
  };
  const validateBody = (value: string) => (value.trim() ? undefined : "Content is required.");
  const validateTags = (value: string) => {
    const duplicates = findDuplicateTags(parseTags(value));
    if (duplicates.length > 0) return `Duplicate tags: ${duplicates.join(", ")}`;
    return undefined;
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
      setMarkdownBody(next);
      markDirty();
      setFieldError("body", validateBody(next));
    } catch (err) {
      console.error(err);
      alert("Failed to upload image.");
    }
  };

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
          fields: "site_language,translation_source_locale,translation_locales",
        });
        const settings = settingsRes.items[0] || {};
        const src = normalizeLocale(String(settings.translation_source_locale || settings.site_language || "ja")) || "ja";
        const targets = parseLocaleList(String(settings.translation_locales || ""));
        return { src, targets };
      } catch {
        return { src: "ja", targets: ["en"] };
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
    const state = location.state as { saved?: boolean; created?: boolean } | null;
    if (!state?.saved) return;
    setError("");
    markSaved(state.created ? "Post saved (new post created)." : "Post saved.");
    navigate(location.pathname, { replace: true });
  }, [location.pathname, location.state, markSaved, navigate]);

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
      ? renderMarkdownToHtml(sourceBody)
      : normalizeMarkdownLinksInHtml(sourceBody);
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
      if (!id || id === "new" || sourcePostId === "") {
        const created = (await pb.collection("posts").create(form)) as unknown as EditorPostRecord;
        markSaved();
        navigate(`/posts/${created.id}`, { state: { saved: true, created: true } });
        return;
      }

      if (selectedLocale === sourceLocale) {
        const updated = (await pb.collection("posts").update(sourcePostId, form)) as unknown as EditorPostRecord;
        setSourceRecord(updated);
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
      {error && <p className="admin-error">{error}</p>}
      {saveMessage && <p className="admin-success">{saveMessage}</p>}
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
          Category
          <input
            value={category}
            onChange={(e) => {
              setCategory(e.target.value);
              markDirty();
              setActiveCategorySuggestion(-1);
            }}
            onKeyDown={onCategoryInputKeyDown}
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
            placeholder="Type tag and press Enter"
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
                    editorMode === "markdown" ? renderMarkdownToHtml(markdownBody) : body,
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
