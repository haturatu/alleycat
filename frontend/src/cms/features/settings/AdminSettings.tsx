import { useEffect, useState, type ReactNode } from "react";
import { ClientResponseError } from "pocketbase";
import { hasRole, pb } from "@cms/lib/pb";
import {
  AdminCheckboxField,
  AdminCheckboxGroupField,
  AdminSelectField,
  AdminTextAreaField,
  AdminTextField,
} from "@cms/ui/AriaControls";
import FormStatusMessage from "@cms/ui/FormStatusMessage";
import SaveButton from "@cms/ui/SaveButton";
import useAdminPageTitle from "@cms/useAdminPageTitle";

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
  enable_ogp_image_generation: false,
  feed_items_limit: 20,
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

function SettingsSection({
  id,
  eyebrow,
  title,
  note,
  children,
}: {
  id: string;
  eyebrow: string;
  title: string;
  note: string;
  children: ReactNode;
}) {
  return (
    <section className="admin-form admin-settings-section" id={id}>
      <div className="admin-settings-section-head">
        <p className="admin-section-label">{eyebrow}</p>
        <h2>{title}</h2>
        <p className="admin-note">{note}</p>
      </div>
      <div className="admin-settings-fields">{children}</div>
    </section>
  );
}

function SettingsSubsection({
  title,
  note,
  children,
}: {
  title: string;
  note: string;
  children: ReactNode;
}) {
  return (
    <div className="admin-settings-subsection">
      <div className="admin-settings-subsection-head">
        <h3>{title}</h3>
        <p className="admin-note">{note}</p>
      </div>
      <div className="admin-settings-subgrid">{children}</div>
    </div>
  );
}

function SettingRow({
  label,
  description,
  control,
}: {
  label: string;
  description: string;
  control: ReactNode;
}) {
  return (
    <div className="admin-setting-row">
      <div className="admin-setting-copy">
        <strong>{label}</strong>
        <p className="admin-note">{description}</p>
      </div>
      <div className="admin-setting-control">{control}</div>
    </div>
  );
}

export default function AdminSettings() {
  const [settings, setSettings] = useState<SettingsRecord>(defaults);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [themeLocked, setThemeLocked] = useState(false);
  const [themeCheckDone, setThemeCheckDone] = useState(false);
  const [geminiApiKey, setGeminiApiKey] = useState("");
  const [hasGeminiApiKey, setHasGeminiApiKey] = useState(false);
  const [error, setError] = useState("");
  const [dirty, setDirty] = useState(false);
  const [lastSavedAt, setLastSavedAt] = useState("");
  const canManageSecrets = hasRole(["admin"]);

  useAdminPageTitle("Settings");

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
        setDirty(false);
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
    setDirty(true);
  };

  const parseLocales = (value: string) =>
    value
      .split(/[,\s;]+/)
      .map((item) => item.trim().toLowerCase())
      .filter(Boolean);

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
      setDirty(false);
      setLastSavedAt(new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }));

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
        setDirty(false);
        setLastSavedAt(new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }));
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
    return <div className="admin-note">Loading settings…</div>;
  }

  return (
    <section>
      <header className="admin-header">
        <div>
          <p className="admin-eyebrow">System Configuration</p>
          <h1>Settings</h1>
        </div>
        <div className="admin-header-actions">
          {saving || dirty || lastSavedAt ? (
            <span className="admin-inline-status">
              {saving ? "Saving…" : dirty ? "Unsaved" : `Saved ${lastSavedAt}`}
            </span>
          ) : null}
          <SaveButton onClick={save} saving={saving} />
        </div>
      </header>
      <div className="admin-settings-nav" aria-label="Settings sections">
        <a href="#settings-foundation">Foundation</a>
        <a href="#settings-translation">Translation</a>
        <a href="#settings-reader">Reader</a>
        <a href="#settings-distribution">Distribution</a>
        <a href="#settings-integrations">Integrations</a>
      </div>
      {dirty ? (
        <div className="admin-unsaved-banner">
          <div>
            <p className="admin-section-label">Unsaved Changes</p>
            <p className="admin-note">Settings have changed. Save when you are ready.</p>
          </div>
          <SaveButton onClick={save} saving={saving} />
        </div>
      ) : null}
      <FormStatusMessage error={error} />
      <div className="admin-settings-overview" aria-label="Settings summary">
        <article className="admin-settings-overview-item">
          <span className="admin-summary-label">Theme</span>
          <strong>{settings.theme}</strong>
        </article>
        <article className="admin-settings-overview-item">
          <span className="admin-summary-label">Translation</span>
          <strong>{settings.enable_post_translation ? "On" : "Off"}</strong>
        </article>
        <article className="admin-settings-overview-item">
          <span className="admin-summary-label">Distribution</span>
          <strong>{settings.enable_feed_xml || settings.enable_feed_json ? "Feeds live" : "Feeds off"}</strong>
        </article>
        <article className="admin-settings-overview-item">
          <span className="admin-summary-label">OGP image</span>
          <strong>{settings.enable_ogp_image_generation ? "On" : "Off"}</strong>
        </article>
      </div>
      <div className="admin-settings-shell">
        <SettingsSection
          id="settings-foundation"
          eyebrow="Identity"
          title="Site Foundation"
          note="Brand, welcome copy, and core presentation settings for the public-facing site."
        >
          <AdminTextField label="Site name" value={settings.site_name} onChange={(value) => update("site_name", value)} />
          <AdminTextField label="Description" value={settings.description} onChange={(value) => update("description", value)} />
          <AdminTextField label="Welcome text" value={settings.welcome_text} onChange={(value) => update("welcome_text", value)} />
          <AdminTextField label="Site URL (origin)" value={settings.site_url} onChange={(value) => update("site_url", value)} placeholder="https://example.com" />
          <AdminTextField label="Site language" value={settings.site_language} onChange={(value) => update("site_language", value)} placeholder="ja" />
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
            <p className="admin-note admin-settings-inline-note">
              Theme selection is disabled because `/frontend/public` has custom assets.
            </p>
          )}
        </SettingsSection>

        <SettingsSection
          id="settings-translation"
          eyebrow="Automation"
          title="Translation Pipeline"
          note="Control source locale, target locales, generation model, and request pacing."
        >
          <SettingRow
            label="Enable post translation"
            description="Generate translation jobs from source posts."
            control={<AdminCheckboxField ariaLabel="Enable post translation" className="admin-check admin-setting-toggle" label="" checked={settings.enable_post_translation} onChange={(checked) => update("enable_post_translation", checked)} />}
          />
          <AdminTextField
            label="Translation source locale"
            value={settings.translation_source_locale}
            onChange={(value) => update("translation_source_locale", value)}
            placeholder="ja"
          />
          <AdminCheckboxGroupField
            ariaLabel="Translation target locales"
            label="Translation target locales"
            values={parseLocales(settings.translation_locales)}
            onChange={(values) => update("translation_locales", values.join(", "))}
            className="admin-settings-locales"
            options={translationLanguageOptions.map((locale) => ({
              value: locale,
              label: locale,
            }))}
          />
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
              onChange={(value) => {
                setGeminiApiKey(value);
                setDirty(true);
              }}
              placeholder={hasGeminiApiKey ? "Saved key is hidden. Leave blank to keep it." : "Paste a new key to save it."}
            />
          )}
        </SettingsSection>

        <SettingsSection
          id="settings-reader"
          eyebrow="Publishing"
          title="Reader Experience"
          note="Tune archive density, excerpts, code presentation, and content scaffolding."
        >
          <div className="admin-settings-fields admin-settings-fields-single">
            <SettingsSubsection title="Archive" note="Controls that shape archive listings and discovery surfaces.">
              <SettingRow
                label="Home page size"
                description="Number of posts shown on the front page."
                control={<AdminTextField ariaLabel="Home page size" label="" type="number" value={String(settings.home_page_size)} onChange={(value) => update("home_page_size", Number(value))} />}
              />
              <SettingRow
                label="Archive page size"
                description="Number of posts shown on archive listing pages."
                control={<AdminTextField ariaLabel="Archive page size" label="" type="number" value={String(settings.archive_page_size)} onChange={(value) => update("archive_page_size", Number(value))} />}
              />
              <SettingRow
                label="Show archive tags"
                description="Display tag groupings on archive pages."
                control={<AdminCheckboxField ariaLabel="Show archive tags" className="admin-check admin-setting-toggle" label="" checked={settings.show_archive_tags} onChange={(checked) => update("show_archive_tags", checked)} />}
              />
              <SettingRow
                label="Show archive search"
                description="Expose search controls inside archive listings."
                control={<AdminCheckboxField ariaLabel="Show archive search" className="admin-check admin-setting-toggle" label="" checked={settings.show_archive_search} onChange={(checked) => update("show_archive_search", checked)} />}
              />
            </SettingsSubsection>
            <SettingsSubsection title="Post page" note="Elements that appear while reading an individual post.">
              <SettingRow
                label="Excerpt length"
                description="Use `0` to keep excerpts manual, or set a generated character count."
                control={<AdminTextField ariaLabel="Excerpt length" label="" type="number" value={String(settings.excerpt_length)} onChange={(value) => update("excerpt_length", Number(value))} />}
              />
              <SettingRow
                label="Show table of contents"
                description="Display a heading index for longer posts."
                control={<AdminCheckboxField ariaLabel="Show table of contents" className="admin-check admin-setting-toggle" label="" checked={settings.show_toc} onChange={(checked) => update("show_toc", checked)} />}
              />
              <SettingRow
                label="Show tags"
                description="Display tags on post pages."
                control={<AdminCheckboxField ariaLabel="Show tags" className="admin-check admin-setting-toggle" label="" checked={settings.show_tags} onChange={(checked) => update("show_tags", checked)} />}
              />
              <SettingRow
                label="Show related posts"
                description="Suggest additional reading at the end of a post."
                control={<AdminCheckboxField ariaLabel="Show related posts" className="admin-check admin-setting-toggle" label="" checked={settings.show_related_posts} onChange={(checked) => update("show_related_posts", checked)} />}
              />
            </SettingsSubsection>
            <SettingsSubsection title="Discovery" note="Taxonomy and navigation aids that help readers browse further.">
              <SettingRow
                label="Show categories"
                description="Expose category labels and navigation paths."
                control={<AdminCheckboxField ariaLabel="Show categories" className="admin-check admin-setting-toggle" label="" checked={settings.show_categories} onChange={(checked) => update("show_categories", checked)} />}
              />
            </SettingsSubsection>
            <SettingsSubsection title="Code display" note="Syntax highlighting and formatting for technical writing.">
              <SettingRow
                label="Enable code highlight"
                description="Apply syntax coloring to fenced code blocks."
                control={<AdminCheckboxField ariaLabel="Enable code highlight" className="admin-check admin-setting-toggle" label="" checked={settings.enable_code_highlight} onChange={(checked) => update("enable_code_highlight", checked)} />}
              />
              <SettingRow
                label="Highlight theme"
                description="Choose the code theme used for highlighted blocks."
                control={
                  <AdminSelectField
                    ariaLabel="Highlight theme"
                    label=""
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
                }
              />
            </SettingsSubsection>
          </div>
        </SettingsSection>

        <SettingsSection
          id="settings-distribution"
          eyebrow="Distribution"
          title="Feeds And Syndication"
          note="Manage feed output and the amount of content exposed to external readers and indexers."
        >
          <SettingRow
            label="Feed items limit"
            description="Maximum number of entries included in feed output."
            control={<AdminTextField ariaLabel="Feed items limit" label="" type="number" value={String(settings.feed_items_limit)} onChange={(value) => update("feed_items_limit", Number(value))} />}
          />
          <SettingRow
            label="Enable RSS/Atom feed"
            description="Expose the XML feed for subscribers and aggregators."
            control={<AdminCheckboxField ariaLabel="Enable RSS/Atom feed" className="admin-check admin-setting-toggle" label="" checked={settings.enable_feed_xml} onChange={(checked) => update("enable_feed_xml", checked)} />}
          />
          <SettingRow
            label="Enable JSON feed"
            description="Expose the JSON Feed endpoint."
            control={<AdminCheckboxField ariaLabel="Enable JSON feed" className="admin-check admin-setting-toggle" label="" checked={settings.enable_feed_json} onChange={(checked) => update("enable_feed_json", checked)} />}
          />
          <SettingRow
            label="Enable OGP image generation"
            description="Generate a share image for post pages even when the post body has no images."
            control={<AdminCheckboxField ariaLabel="Enable OGP image generation" className="admin-check admin-setting-toggle" label="" checked={settings.enable_ogp_image_generation} onChange={(checked) => update("enable_ogp_image_generation", checked)} />}
          />
        </SettingsSection>

        <SettingsSection
          id="settings-integrations"
          eyebrow="Integrations"
          title="Analytics, Ads, And Comments"
          note="Third-party scripts and measurement tools live here so they stay separate from editorial controls."
        >
          <SettingRow
            label="Enable analytics"
            description="Allow analytics scripts on the public site."
            control={<AdminCheckboxField ariaLabel="Enable analytics" className="admin-check admin-setting-toggle" label="" checked={settings.enable_analytics} onChange={(checked) => update("enable_analytics", checked)} />}
          />
          <AdminTextField label="Analytics URL" value={settings.analytics_url} onChange={(value) => update("analytics_url", value)} />
          <AdminTextField label="Analytics site id" value={settings.analytics_site_id} onChange={(value) => update("analytics_site_id", value)} />
          <SettingRow
            label="Enable ads"
            description="Load ad scripts and ad placements."
            control={<AdminCheckboxField ariaLabel="Enable ads" className="admin-check admin-setting-toggle" label="" checked={settings.enable_ads} onChange={(checked) => update("enable_ads", checked)} />}
          />
          <AdminTextField label="Ads client" value={settings.ads_client} onChange={(value) => update("ads_client", value)} />
          <SettingRow
            label="Enable comments"
            description="Render embedded comment threads on posts."
            control={<AdminCheckboxField ariaLabel="Enable comments" className="admin-check admin-setting-toggle" label="" checked={settings.enable_comments} onChange={(checked) => update("enable_comments", checked)} />}
          />
          <AdminTextAreaField
            label="Comment script tag (utterances/giscus)"
            value={settings.comments_script_tag}
            onChange={(value) => update("comments_script_tag", value)}
            rows={4}
            placeholder={`<script src="https://utteranc.es/client.js" repo="owner/repo" issue-term="pathname" theme="github-dark" crossorigin="anonymous" async></script>`}
          />
        </SettingsSection>
      </div>
    </section>
  );
}
