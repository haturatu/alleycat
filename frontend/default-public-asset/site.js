(() => {
  const root = document.documentElement;
  const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)").matches;
  let theme = localStorage.getItem("theme") || (prefersDark ? "dark" : "light");

  const applyTheme = (nextTheme) => {
    root.dataset.theme = nextTheme;
    const hljsThemeLink = document.getElementById("hljs-theme-link");
    if (!hljsThemeLink) return;
    const href = nextTheme === "dark"
      ? hljsThemeLink.getAttribute("data-theme-dark")
      : hljsThemeLink.getAttribute("data-theme-light");
    if (href) hljsThemeLink.setAttribute("href", href);
  };

  applyTheme(theme);
  document.querySelector("[data-theme-toggle]")?.addEventListener("click", () => {
    theme = theme === "dark" ? "light" : "dark";
    localStorage.setItem("theme", theme);
    applyTheme(theme);
  });

  const highlight = () => window.hljs?.highlightAll();
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", highlight, { once: true });
  } else {
    highlight();
  }
})();
