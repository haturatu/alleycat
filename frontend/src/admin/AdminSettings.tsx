import { useEffect, useState } from "react";
import { ClientResponseError } from "pocketbase";
import { hasRole, pb } from "../lib/pb";
import {
  AdminButton,
  AdminCheckboxField,
  AdminSelectField,
  AdminTextAreaField,
  AdminTextField,
} from "./components/AriaControls";
import SaveButton from "./components/SaveButton";

const translationLanguageOptions = [
  "en",
  "ja",
  "zh-cn",
  "zh-tw",
  "ko",
  "fr",
  "de",
  "es",
  "hi",
  "ar",
  "bn",
  "pt",
  "ru",
];

const defaults = {
  site_name: "Example Blog",
  description: "A calm place to write.",
  welcome_text: "Welcome to your blog",
  home_top_image: "/default-hero.svg",
  home_top_image_alt: "Default hero image",
  footer_html: "",
  theme: "ember",
  site_url: "",
  site_language: "ja",
  enable_feed_xml: true,
  enable_feed_json: true,
  feed_items_limit: 30,
  enable_analytics: false,
  analytics_url: "",
  analytics_site_id: "",
  enable_ads: false,
  ads_client: "",
  enable_comments: false,
  comments_script_tag: "",
  enable_code_highlight: true,
  highlight_theme: "github-dark",
  archive_page_size: 10,
  excerpt_length: 0,
  home_page_size: 3,
  show_toc: true,
  show_archive_tags: true,
  show_tags: true,
  show_categories: true,
  show_related_posts: false,
  show_archive_search: true,
  enable_post_translation: false,
  translation_source_locale: "ja",
  translation_locales: "en",
  translation_model: "gemini-1.5-flash",
  translation_requests_per_minute: 60,
};

type SettingsRecord = typeof defaults & { id?: string };

export default function AdminSettings() {
  const [settings, setSettings] = useState<SettingsRecord>(defaults);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [themeLocked, setThemeLocked] = useState(false);
  const [themeCheckDone, setThemeCheckDone] = useState(false);
  const [geminiApiKey, setGeminiApiKey] = useState("");
  const [hasGeminiApiKey, setHasGeminiApiKey] = useState(false);
  const [error, setError] = useState("");
  const canManageSecrets = hasRole(["admin"]);

  useEffect(() => {
    const load = async () => {
      try {
        const res = await pb.collection("settings").getList(1, 1);
        if (res.items.length > 0) {
          const merged = { ...defaults, ...res.items[0] };
          setSettings(merged);
        } else {
          setSettings(defaults);
        }
      } catch (err) {
        console.error("settings load failed", err);
        setSettings(defaults);
      } finally {
        if (canManageSecrets) {
          try {
            const secretRes = await pb.collection("app_secrets").getList(1, 1, { fields: "id,gemini_api_key" });
            const storedKey = String(secretRes.items[0]?.gemini_api_key || "").trim();
            setHasGeminiApiKey(storedKey !== "");
          } catch {
            setHasGeminiApiKey(false);
          }
        }
        setLoading(false);
      }
    };
    load();
  }, [canManageSecrets]);

  useEffect(() => {
    const check = async () => {
      try {
        const res = await fetch("/theme-status");
        if (res.ok) {
          const data = await res.json();
          setThemeLocked(Boolean(data?.publicAssets));
        }
      } catch {
        // ignore
      } finally {
        setThemeCheckDone(true);
      }
    };
    check();
  }, []);

  const update = (key: keyof SettingsRecord, value: string | number | boolean) => {
    setSettings((prev) => ({ ...prev, [key]: value }));
  };

  const parseLocales = (value: string) =>
    value
      .split(/[,\s;]+/)
      .map((item) => item.trim().toLowerCase())
      .filter(Boolean);

  const toggleLocale = (locale: string) => {
    const current = new Set(parseLocales(settings.translation_locales));
    if (current.has(locale)) {
      current.delete(locale);
    } else {
      current.add(locale);
    }
    update("translation_locales", Array.from(current).join(", "));
  };

  const save = async () => {
    setError("");
    setSaving(true);
    try {
      let settingsId = settings.id;
      if (!settingsId) {
        const res = await pb.collection("settings").getList(1, 1, { fields: "id" });
        settingsId = res.items[0]?.id;
      }
      if (!settingsId) {
        setError("Settings record is not initialized yet. Please restart PocketBase.");
        return;
      }
      const payload = {
        ...settings,
        translation_source_locale: settings.translation_source_locale.trim().toLowerCase(),
        translation_locales: settings.translation_locales.trim().toLowerCase(),
        translation_model: settings.translation_model.trim(),
        translation_requests_per_minute: Number(settings.translation_requests_per_minute) || 60,
      };
      delete payload.id;
      const updated = await pb.collection("settings").update(settingsId, payload);
      setSettings({ ...defaults, ...updated });

      const trimmedGeminiKey = geminiApiKey.trim();
      if (canManageSecrets && trimmedGeminiKey !== "") {
        const secretRes = await pb.collection("app_secrets").getList(1, 1, { fields: "id" });
        if (secretRes.items.length > 0) {
          await pb.collection("app_secrets").update(secretRes.items[0].id, {
            gemini_api_key: trimmedGeminiKey,
          });
        } else {
          await pb.collection("app_secrets").create({
            gemini_api_key: trimmedGeminiKey,
          });
        }
        setGeminiApiKey("");
        setHasGeminiApiKey(true);
      }
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
      console.error("settings save failed", err);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <section>
      <header className="admin-header">
        <h1>Settings</h1>
        <SaveButton onClick={save} saving={saving} />
      </header>
      {error && <p className="admin-error">{error}</p>}
      <div className="admin-form">
        <AdminTextField label="Site name" value={settings.site_name} onChange={(value) => update("site_name", value)} />
        <AdminTextField label="Description" value={settings.description} onChange={(value) => update("description", value)} />
        <AdminTextField label="Welcome text" value={settings.welcome_text} onChange={(value) => update("welcome_text", value)} />
        <AdminTextField label="Home top image" value={settings.home_top_image} onChange={(value) => update("home_top_image", value)} />
        <AdminTextField label="Home top image alt" value={settings.home_top_image_alt} onChange={(value) => update("home_top_image_alt", value)} />
        <AdminTextAreaField label="Footer HTML" value={settings.footer_html} onChange={(value) => update("footer_html", value)} rows={3} />
        <AdminSelectField
          label="Theme"
          value={settings.theme}
          onChange={(value) => update("theme", value)}
          disabled={themeLocked}
          options={[
            { value: "ember", label: "Ember (default)" },
            { value: "terminal", label: "Terminal" },
            { value: "wiki", label: "Wiki" },
            { value: "docs", label: "Docs" },
            { value: "minimal", label: "Minimal" },
          ]}
        />
        {themeCheckDone && themeLocked && (
          <p className="admin-note">Theme selection is disabled because /frontend/public has assets.</p>
        )}
        <AdminTextField label="Site URL (feeds)" value={settings.site_url} onChange={(value) => update("site_url", value)} placeholder="https://example.com" />
        <AdminTextField label="Site language" value={settings.site_language} onChange={(value) => update("site_language", value)} placeholder="ja" />
        <AdminCheckboxField label="Enable post translation" checked={settings.enable_post_translation} onChange={(checked) => update("enable_post_translation", checked)} />
        <AdminTextField
          label="Translation source locale"
          value={settings.translation_source_locale}
          onChange={(value) => update("translation_source_locale", value)}
          placeholder="ja"
        />
        <div className="admin-field">
          <span>Translation target locales</span>
          <div className="admin-tag-suggestions">
            {translationLanguageOptions.map((locale) => {
              const selected = parseLocales(settings.translation_locales).includes(locale);
              return (
                <AdminButton
                  key={locale}
                  onPress={() => toggleLocale(locale)}
                  style={{ opacity: selected ? 1 : 0.6 }}
                >
                  {locale}
                </AdminButton>
              );
            })}
          </div>
        </div>
        <AdminTextField label="Translation locales (comma separated)" value={settings.translation_locales} onChange={(value) => update("translation_locales", value)} placeholder="en, zh-cn" />
        <AdminTextField label="Gemini model" value={settings.translation_model} onChange={(value) => update("translation_model", value)} placeholder="gemini-1.5-flash" />
        <AdminTextField
          label="Translation requests/minute"
          type="number"
          value={String(settings.translation_requests_per_minute)}
          onChange={(value) => update("translation_requests_per_minute", Number(value))}
          min={1}
          max={1000}
        />
        {canManageSecrets && (
          <AdminTextField
            label={`Gemini API Key ${hasGeminiApiKey ? "(saved)" : "(not set)"}`}
            type="password"
            value={geminiApiKey}
            onChange={setGeminiApiKey}
            placeholder="Leave blank to keep current key"
          />
        )}
        <AdminTextField label="Feed items limit" type="number" value={String(settings.feed_items_limit)} onChange={(value) => update("feed_items_limit", Number(value))} />
        <AdminCheckboxField label="Enable RSS/Atom feed" checked={settings.enable_feed_xml} onChange={(checked) => update("enable_feed_xml", checked)} />
        <AdminCheckboxField label="Enable JSON feed" checked={settings.enable_feed_json} onChange={(checked) => update("enable_feed_json", checked)} />
        <AdminCheckboxField label="Enable code highlight" checked={settings.enable_code_highlight} onChange={(checked) => update("enable_code_highlight", checked)} />
        <AdminSelectField
          label="Highlight theme"
          value={settings.highlight_theme}
          onChange={(value) => update("highlight_theme", value)}
          options={[
            { value: "github-dark", label: "github-dark" },
            { value: "github", label: "github" },
            { value: "atom-one-dark", label: "atom-one-dark" },
            { value: "atom-one-light", label: "atom-one-light" },
            { value: "monokai", label: "monokai" },
            { value: "tokyo-night-dark", label: "tokyo-night-dark" },
            { value: "tokyo-night-light", label: "tokyo-night-light" },
            { value: "solarized-dark", label: "solarized-dark" },
            { value: "solarized-light", label: "solarized-light" },
            { value: "dracula", label: "dracula" },
            { value: "vs", label: "vs" },
          ]}
        />
        <AdminTextField label="Home page size" type="number" value={String(settings.home_page_size)} onChange={(value) => update("home_page_size", Number(value))} />
        <AdminTextField label="Archive page size" type="number" value={String(settings.archive_page_size)} onChange={(value) => update("archive_page_size", Number(value))} />
        <AdminTextField label="Excerpt length (0 = manual)" type="number" value={String(settings.excerpt_length)} onChange={(value) => update("excerpt_length", Number(value))} />
        <AdminCheckboxField label="Show table of contents" checked={settings.show_toc} onChange={(checked) => update("show_toc", checked)} />
        <AdminCheckboxField label="Show archive tags" checked={settings.show_archive_tags} onChange={(checked) => update("show_archive_tags", checked)} />
        <AdminCheckboxField label="Show tags" checked={settings.show_tags} onChange={(checked) => update("show_tags", checked)} />
        <AdminCheckboxField label="Show categories" checked={settings.show_categories} onChange={(checked) => update("show_categories", checked)} />
        <AdminCheckboxField label="Show related posts" checked={settings.show_related_posts} onChange={(checked) => update("show_related_posts", checked)} />
        <AdminCheckboxField label="Show archive search slot" checked={settings.show_archive_search} onChange={(checked) => update("show_archive_search", checked)} />
        <AdminCheckboxField label="Enable analytics" checked={settings.enable_analytics} onChange={(checked) => update("enable_analytics", checked)} />
        <AdminTextField label="Analytics URL" value={settings.analytics_url} onChange={(value) => update("analytics_url", value)} />
        <AdminTextField label="Analytics site id" value={settings.analytics_site_id} onChange={(value) => update("analytics_site_id", value)} />
        <AdminCheckboxField label="Enable ads" checked={settings.enable_ads} onChange={(checked) => update("enable_ads", checked)} />
        <AdminTextField label="Ads client" value={settings.ads_client} onChange={(value) => update("ads_client", value)} />
        <AdminCheckboxField label="Enable comments" checked={settings.enable_comments} onChange={(checked) => update("enable_comments", checked)} />
        <AdminTextAreaField
          label="Comment script tag (utterances/giscus)"
          value={settings.comments_script_tag}
          onChange={(value) => update("comments_script_tag", value)}
          rows={4}
          placeholder={`<script src="https://utteranc.es/client.js" repo="owner/repo" issue-term="pathname" theme="github-dark" crossorigin="anonymous" async></script>`}
        />
      </div>
    </section>
  );
}
