import { useEffect, useMemo, useState, type KeyboardEvent } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { ClientResponseError } from "pocketbase";
import { pb } from "../lib/pb";
import { buildExcerpt, normalizeMarkdownLinksInHtml, parseTags, slugify } from "../utils/text";
import RichEditor from "./RichEditor";

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
  const [saveMessage, setSaveMessage] = useState("");
  const [saving, setSaving] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [slugEditedManually, setSlugEditedManually] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [activeTagSuggestion, setActiveTagSuggestion] = useState(-1);
  const [activeCategorySuggestion, setActiveCategorySuggestion] = useState(-1);
  const [excerptLength, setExcerptLength] = useState(0);

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

  const applyRecordToForm = (record: EditorPostRecord) => {
    setError("");
    setTitle(record.title || "");
    setSlug(record.slug || "");
    setBody(record.body || "");
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
    setIsDirty(false);
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
        setIsDirty(false);
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
        setError("記事の取得に失敗しました。権限またはIDを確認してください。");
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

  useEffect(() => {
    if (!isDirty) return;
    const beforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", beforeUnload);
    return () => window.removeEventListener("beforeunload", beforeUnload);
  }, [isDirty]);

  useEffect(() => {
    if (!isDirty) return;
    const onClickCapture = (event: MouseEvent) => {
      const target = event.target as HTMLElement | null;
      const anchor = target?.closest?.("a[href]") as HTMLAnchorElement | null;
      if (!anchor) return;
      if (anchor.target === "_blank" || anchor.hasAttribute("download")) return;
      const href = anchor.getAttribute("href");
      if (!href || href.startsWith("#")) return;
      const next = new URL(anchor.href, window.location.href);
      if (next.origin !== window.location.origin) return;
      const currentPath = `${window.location.pathname}${window.location.search}${window.location.hash}`;
      const nextPath = `${next.pathname}${next.search}${next.hash}`;
      if (currentPath === nextPath) return;
      if (window.confirm("You have unsaved changes. Leave without saving?")) return;
      event.preventDefault();
      event.stopPropagation();
    };
    document.addEventListener("click", onClickCapture, true);
    return () => document.removeEventListener("click", onClickCapture, true);
  }, [isDirty]);

  useEffect(() => {
    const state = location.state as { saved?: boolean; created?: boolean } | null;
    if (!state?.saved) return;
    setError("");
    setSaveMessage(state.created ? "Post saved (new post created)." : "Post saved.");
    navigate(location.pathname, { replace: true });
  }, [location.pathname, location.state, navigate]);

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
    setSaveMessage("");
    setIsDirty(true);
    setActiveTagSuggestion(-1);
    setFieldError("tags", validateTags(merged));
  };

  const removeTag = (tag: string) => {
    const merged = currentTags
      .filter((item) => item.toLowerCase() !== tag.toLowerCase())
      .join(", ");
    setTags(merged);
    setSaveMessage("");
    setIsDirty(true);
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
      setSaveMessage("");
      setIsDirty(true);
      setActiveCategorySuggestion(-1);
    }
  };

  const save = async () => {
    if (saving) return;
    setError("");
    setSaveMessage("");
    const normalizedBody = normalizeMarkdownLinksInHtml(body);
    const autoExcerpt = excerptLength > 0;
    const finalExcerpt = autoExcerpt ? buildExcerpt(normalizedBody, excerptLength) : excerpt;
    const trimmedTitle = title.trim();
    const trimmedSlug = slug.trim();
    const trimmedBody = normalizedBody.trim();
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
        setIsDirty(false);
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
      setIsDirty(false);
      setSaveMessage("Post saved.");
    } catch (err) {
      if (err instanceof ClientResponseError) {
        const details = err.response?.data as Record<string, { message?: string }> | undefined;
        const detailText = details
          ? Object.entries(details)
              .map(([field, value]) => `${field}: ${value?.message || "invalid"}`)
              .join(", ")
          : "";
        setError(detailText ? `保存に失敗しました: ${detailText}` : "保存に失敗しました。");
      } else {
        setError("保存に失敗しました。");
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
        <button className="admin-primary" onClick={save} disabled={saving}>
          {saving ? "Saving..." : "Save"}
        </button>
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
              setSaveMessage("");
              setIsDirty(true);
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
                setSaveMessage("");
                setIsDirty(true);
                setFieldError("slug", validateSlug(next));
              }}
            />
            <button
              type="button"
              onClick={() => {
                const auto = slugify(title);
                setSlug(auto);
                setSlugEditedManually(false);
                setSaveMessage("");
                setIsDirty(true);
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
              setSaveMessage("");
              setIsDirty(true);
            }}
          />
        </label>
        <label>
          Category
          <input
            value={category}
            onChange={(e) => {
              setCategory(e.target.value);
              setSaveMessage("");
              setIsDirty(true);
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
                  setSaveMessage("");
                  setIsDirty(true);
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
              setSaveMessage("");
              setIsDirty(true);
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
              setSaveMessage("");
              setIsDirty(true);
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
              setSaveMessage("");
              setIsDirty(true);
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
              setSaveMessage("");
              setIsDirty(true);
            }}
          />
        </label>
        <label>
          Excerpt
          <textarea
            value={excerptLength > 0 ? buildExcerpt(body, excerptLength) : excerpt}
            onChange={(e) => {
              setExcerpt(e.target.value);
              setSaveMessage("");
              setIsDirty(true);
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
              setSaveMessage("");
              setIsDirty(true);
            }}
          />
        </label>
        <div className="admin-field">
          <span>Content</span>
          <RichEditor
            value={body}
            onChange={(value) => {
              setBody(value);
              setSaveMessage("");
              setIsDirty(true);
              setFieldError("body", validateBody(value));
            }}
          />
        </div>
        {fieldErrors.body && <p className="admin-error-inline">{fieldErrors.body}</p>}
      </div>
    </section>
  );
}
