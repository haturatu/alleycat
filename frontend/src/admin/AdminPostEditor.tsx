import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ClientResponseError } from "pocketbase";
import { pb } from "../lib/pb";
import { buildExcerpt, parseTags, slugify } from "../utils/text";
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
  const navigate = useNavigate();
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [body, setBody] = useState("");
  const [excerpt, setExcerpt] = useState("");
  const [tags, setTags] = useState("");
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
  const [excerptLength, setExcerptLength] = useState(0);

  const [sourcePostId, setSourcePostId] = useState("");
  const [sourceLocale, setSourceLocale] = useState("ja");
  const [selectedLocale, setSelectedLocale] = useState("ja");
  const [localeOptions, setLocaleOptions] = useState<string[]>(["ja"]);
  const [sourceRecord, setSourceRecord] = useState<EditorPostRecord | null>(null);
  const [localeRecords, setLocaleRecords] = useState<Record<string, EditorPostTranslationRecord>>({});

  const applyRecordToForm = (record: EditorPostRecord) => {
    setError("");
    setTitle(record.title || "");
    setSlug(record.slug || "");
    setBody(record.body || "");
    setExcerpt(record.excerpt || "");
    setTags(record.tags || "");
    setCategory(record.category || "");
    setAuthor(record.author || "");
    setPublishedAt(record.published_at ? record.published_at.slice(0, 16) : "");
    setPublished(Boolean(record.published));
    setFeaturedImage(null);
    setAttachments([]);
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

  const save = async () => {
    setError("");
    const autoExcerpt = excerptLength > 0;
    const finalExcerpt = autoExcerpt ? buildExcerpt(body, excerptLength) : excerpt;
    const trimmedTitle = title.trim();
    const trimmedSlug = slug.trim();
    const trimmedBody = body.trim();
    if (!trimmedTitle || !trimmedSlug || !trimmedBody) {
      setError("Title / Slug / Content は必須です。");
      return;
    }

    const form = new FormData();
    form.set("title", trimmedTitle);
    form.set("slug", trimmedSlug);
    form.set("body", body);
    form.set("excerpt", (finalExcerpt || buildExcerpt(body)).trim());
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

    try {
      if (!id || id === "new" || sourcePostId === "") {
        const created = (await pb.collection("posts").create(form)) as unknown as EditorPostRecord;
        navigate(`/posts/${created.id}`);
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
    }
  };

  return (
    <section>
      <header className="admin-header">
        <h1>{id === "new" ? "New Post" : "Edit Post"}</h1>
        <button className="admin-primary" onClick={save}>
          Save
        </button>
      </header>
      {error && <p className="admin-error">{error}</p>}
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
          <input value={title} onChange={(e) => setTitle(e.target.value)} />
        </label>
        <label>
          Slug
          <div className="admin-inline">
            <input value={slug} onChange={(e) => setSlug(e.target.value)} />
            <button type="button" onClick={() => setSlug(slugify(title))}>
              Auto
            </button>
          </div>
        </label>
        <label>
          Published at
          <input
            type="datetime-local"
            value={publishedAt}
            onChange={(e) => setPublishedAt(e.target.value)}
          />
        </label>
        <label>
          Category
          <input
            list="category-list"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          />
        </label>
        <datalist id="category-list">
          {categories.map((item) => (
            <option key={item} value={item} />
          ))}
        </datalist>
        <label>
          Author
          <select value={author} onChange={(e) => setAuthor(e.target.value)}>
            <option value="">(none)</option>
            {authors.map((user) => (
              <option key={user.id} value={user.id}>
                {user.name || user.email || user.id}
              </option>
            ))}
          </select>
        </label>
        <label>
          Tags (comma separated)
          <input value={tags} onChange={(e) => setTags(e.target.value)} />
        </label>
        {tagOptions.length > 0 && (
          <div className="admin-tag-suggestions">
            {tagOptions
              .filter((tag) => !parseTags(tags).includes(tag))
              .slice(0, 20)
              .map((tag) => (
                <button
                  type="button"
                  key={tag}
                  onClick={() => {
                    const next = [...parseTags(tags), tag].join(", ");
                    setTags(next);
                  }}
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
            onChange={(e) => setFeaturedImage(e.target.files ? e.target.files[0] : null)}
          />
        </label>
        <label>
          Attachments
          <input
            type="file"
            multiple
            onChange={(e) =>
              setAttachments(e.target.files ? Array.from(e.target.files) : [])
            }
          />
        </label>
        <label>
          Excerpt
          <textarea
            value={excerptLength > 0 ? buildExcerpt(body, excerptLength) : excerpt}
            onChange={(e) => setExcerpt(e.target.value)}
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
          <input type="checkbox" checked={published} onChange={(e) => setPublished(e.target.checked)} />
        </label>
        <div className="admin-field">
          <span>Content</span>
          <RichEditor value={body} onChange={setBody} />
        </div>
      </div>
    </section>
  );
}
