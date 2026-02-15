import { useEffect, useState } from "react";
import { ClientResponseError } from "pocketbase";
import { hasRole, pb } from "../lib/pb";

const translationLanguageOptions = ["en", "ja", "zh-cn", "zh-tw", "ko", "fr", "de", "es"];

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
  enable_code_highlight: true,
  highlight_theme: "github-dark",
  archive_page_size: 10,
  excerpt_length: 0,
  home_page_size: 3,
  show_toc: true,
  show_archive_tags: true,
  show_tags: true,
  show_categories: true,
  show_archive_search: true,
  enable_post_translation: false,
  translation_source_locale: "ja",
  translation_locales: "en",
  translation_model: "gemini-1.5-flash",
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
        <button className="admin-primary" onClick={save} disabled={saving}>
          {saving ? "Saving..." : "Save"}
        </button>
      </header>
      {error && <p className="admin-error">{error}</p>}
      <div className="admin-form">
        <label>
          Site name
          <input value={settings.site_name} onChange={(e) => update("site_name", e.target.value)} />
        </label>
        <label>
          Description
          <input value={settings.description} onChange={(e) => update("description", e.target.value)} />
        </label>
        <label>
          Welcome text
          <input value={settings.welcome_text} onChange={(e) => update("welcome_text", e.target.value)} />
        </label>
        <label>
          Home top image
          <input value={settings.home_top_image} onChange={(e) => update("home_top_image", e.target.value)} />
        </label>
        <label>
          Home top image alt
          <input value={settings.home_top_image_alt} onChange={(e) => update("home_top_image_alt", e.target.value)} />
        </label>
        <label>
          Footer HTML
          <textarea value={settings.footer_html} onChange={(e) => update("footer_html", e.target.value)} rows={3} />
        </label>
        <label>
          Theme
          <select
            value={settings.theme}
            onChange={(e) => update("theme", e.target.value)}
            disabled={themeLocked}
          >
            <option value="ember">Ember (default)</option>
            <option value="terminal">Terminal</option>
            <option value="wiki">Wiki</option>
            <option value="docs">Docs</option>
            <option value="minimal">Minimal</option>
          </select>
        </label>
        {themeCheckDone && themeLocked && (
          <p className="admin-note">Theme selection is disabled because /frontend/public has assets.</p>
        )}
        <label>
          Site URL (feeds)
          <input value={settings.site_url} onChange={(e) => update("site_url", e.target.value)} placeholder="https://example.com" />
        </label>
        <label>
          Site language
          <input value={settings.site_language} onChange={(e) => update("site_language", e.target.value)} placeholder="ja" />
        </label>
        <label className="admin-check admin-check-right">
          <span>Enable post translation</span>
          <input
            type="checkbox"
            checked={settings.enable_post_translation}
            onChange={(e) => update("enable_post_translation", e.target.checked)}
          />
        </label>
        <label>
          Translation source locale
          <input
            value={settings.translation_source_locale}
            onChange={(e) => update("translation_source_locale", e.target.value)}
            placeholder="ja"
          />
        </label>
        <div className="admin-field">
          <span>Translation target locales</span>
          <div className="admin-tag-suggestions">
            {translationLanguageOptions.map((locale) => {
              const selected = parseLocales(settings.translation_locales).includes(locale);
              return (
                <button
                  type="button"
                  key={locale}
                  onClick={() => toggleLocale(locale)}
                  style={{ opacity: selected ? 1 : 0.6 }}
                >
                  {locale}
                </button>
              );
            })}
          </div>
        </div>
        <label>
          Translation locales (comma separated)
          <input
            value={settings.translation_locales}
            onChange={(e) => update("translation_locales", e.target.value)}
            placeholder="en, zh-cn"
          />
        </label>
        <label>
          Gemini model
          <input
            value={settings.translation_model}
            onChange={(e) => update("translation_model", e.target.value)}
            placeholder="gemini-1.5-flash"
          />
        </label>
        {canManageSecrets && (
          <label>
            Gemini API Key {hasGeminiApiKey ? "(saved)" : "(not set)"}
            <input
              type="password"
              value={geminiApiKey}
              onChange={(e) => setGeminiApiKey(e.target.value)}
              placeholder="Leave blank to keep current key"
            />
          </label>
        )}
        <label>
          Feed items limit
          <input
            type="number"
            value={settings.feed_items_limit}
            onChange={(e) => update("feed_items_limit", Number(e.target.value))}
          />
        </label>
        <label className="admin-check admin-check-right">
          <span>Enable RSS/Atom feed</span>
          <input
            type="checkbox"
            checked={settings.enable_feed_xml}
            onChange={(e) => update("enable_feed_xml", e.target.checked)}
          />
        </label>
        <label className="admin-check admin-check-right">
          <span>Enable JSON feed</span>
          <input
            type="checkbox"
            checked={settings.enable_feed_json}
            onChange={(e) => update("enable_feed_json", e.target.checked)}
          />
        </label>
        <label className="admin-check admin-check-right">
          <span>Enable code highlight</span>
          <input
            type="checkbox"
            checked={settings.enable_code_highlight}
            onChange={(e) => update("enable_code_highlight", e.target.checked)}
          />
        </label>
        <label>
          Highlight theme
          <select
            value={settings.highlight_theme}
            onChange={(e) => update("highlight_theme", e.target.value)}
          >
            <option value="github-dark">github-dark</option>
            <option value="github">github</option>
            <option value="atom-one-dark">atom-one-dark</option>
            <option value="atom-one-light">atom-one-light</option>
            <option value="monokai">monokai</option>
            <option value="tokyo-night-dark">tokyo-night-dark</option>
            <option value="tokyo-night-light">tokyo-night-light</option>
            <option value="solarized-dark">solarized-dark</option>
            <option value="solarized-light">solarized-light</option>
            <option value="dracula">dracula</option>
            <option value="vs">vs</option>
          </select>
        </label>
        <label>
          Home page size
          <input
            type="number"
            value={settings.home_page_size}
            onChange={(e) => update("home_page_size", Number(e.target.value))}
          />
        </label>
        <label>
          Archive page size
          <input
            type="number"
            value={settings.archive_page_size}
            onChange={(e) => update("archive_page_size", Number(e.target.value))}
          />
        </label>
        <label>
          Excerpt length (0 = manual)
          <input
            type="number"
            value={settings.excerpt_length}
            onChange={(e) => update("excerpt_length", Number(e.target.value))}
          />
        </label>
        <label className="admin-check admin-check-right">
          <span>Show table of contents</span>
          <input
            type="checkbox"
            checked={settings.show_toc}
            onChange={(e) => update("show_toc", e.target.checked)}
          />
        </label>
        <label className="admin-check admin-check-right">
          <span>Show archive tags</span>
          <input
            type="checkbox"
            checked={settings.show_archive_tags}
            onChange={(e) => update("show_archive_tags", e.target.checked)}
          />
        </label>
        <label className="admin-check admin-check-right">
          <span>Show tags</span>
          <input
            type="checkbox"
            checked={settings.show_tags}
            onChange={(e) => update("show_tags", e.target.checked)}
          />
        </label>
        <label className="admin-check admin-check-right">
          <span>Show categories</span>
          <input
            type="checkbox"
            checked={settings.show_categories}
            onChange={(e) => update("show_categories", e.target.checked)}
          />
        </label>
        <label className="admin-check admin-check-right">
          <span>Show archive search slot</span>
          <input
            type="checkbox"
            checked={settings.show_archive_search}
            onChange={(e) => update("show_archive_search", e.target.checked)}
          />
        </label>
        <label className="admin-check admin-check-right">
          <span>Enable analytics</span>
          <input
            type="checkbox"
            checked={settings.enable_analytics}
            onChange={(e) => update("enable_analytics", e.target.checked)}
          />
        </label>
        <label>
          Analytics URL
          <input value={settings.analytics_url} onChange={(e) => update("analytics_url", e.target.value)} />
        </label>
        <label>
          Analytics site id
          <input value={settings.analytics_site_id} onChange={(e) => update("analytics_site_id", e.target.value)} />
        </label>
        <label className="admin-check admin-check-right">
          <span>Enable ads</span>
          <input
            type="checkbox"
            checked={settings.enable_ads}
            onChange={(e) => update("enable_ads", e.target.checked)}
          />
        </label>
        <label>
          Ads client
          <input value={settings.ads_client} onChange={(e) => update("ads_client", e.target.value)} />
        </label>
      </div>
    </section>
  );
}
