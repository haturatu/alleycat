import { useEffect, useState } from "react";
import { pb } from "../lib/pb";

const defaults = {
  site_name: "Example Blog",
  description: "A calm place to write.",
  welcome_text: "Welcome to your blog",
  home_top_image: "/default-hero.svg",
  home_top_image_alt: "Default hero image",
  footer_html: "",
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
  home_page_size: 3,
  show_toc: true,
  show_archive_tags: true,
  show_archive_search: true,
};

type SettingsRecord = typeof defaults & { id?: string };

export default function AdminSettings() {
  const [settings, setSettings] = useState<SettingsRecord>(defaults);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const load = async () => {
      try {
        const res = await pb.collection("settings").getList(1, 1);
        if (res.items.length > 0) {
          const merged = { ...defaults, ...res.items[0] };
          setSettings(merged);
        } else {
          const created = await pb.collection("settings").create(defaults);
          setSettings({ ...defaults, ...created });
        }
      } catch {
        setSettings(defaults);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  const update = (key: keyof SettingsRecord, value: string | number | boolean) => {
    setSettings((prev) => ({ ...prev, [key]: value }));
  };

  const save = async () => {
    if (!settings.id) return;
    setSaving(true);
    try {
      const payload = {
        ...settings,
      };
      delete payload.id;
      const updated = await pb.collection("settings").update(settings.id, payload);
      setSettings({ ...defaults, ...updated });
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
          Site URL (feeds)
          <input value={settings.site_url} onChange={(e) => update("site_url", e.target.value)} placeholder="https://example.com" />
        </label>
        <label>
          Site language
          <input value={settings.site_language} onChange={(e) => update("site_language", e.target.value)} placeholder="ja" />
        </label>
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
