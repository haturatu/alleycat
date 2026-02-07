import { useEffect, useMemo, useState } from "react";
import { Link, Outlet, useLocation } from "react-router-dom";
import { pb, PageRecord } from "../lib/pb";
import { siteConfig } from "../config";

const getTheme = () => {
  const stored = localStorage.getItem("theme");
  if (stored) return stored;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
};

export default function Layout() {
  const location = useLocation();
  const [menuPages, setMenuPages] = useState<PageRecord[]>([]);
  const [theme, setTheme] = useState("light");

  useEffect(() => {
    const nextTheme = getTheme();
    setTheme(nextTheme);
    document.documentElement.dataset.theme = nextTheme;
  }, []);

  useEffect(() => {
    pb.collection("pages")
      .getFullList<PageRecord>({
        filter: "menuVisible = true && published = true",
        sort: "menuOrder",
      })
      .then(setMenuPages)
      .catch(() => setMenuPages([]));
  }, []);

  const toggleTheme = () => {
    const nextTheme = theme === "dark" ? "light" : "dark";
    localStorage.setItem("theme", nextTheme);
    document.documentElement.dataset.theme = nextTheme;
    setTheme(nextTheme);
  };

  const menuLinks = useMemo(() => siteConfig.menuLinks, []);

  const isPost = location.pathname.startsWith("/posts");
  const bodyClass = location.pathname === "/" ? "body-home" : isPost ? "body-post" : "body-tag";

  return (
    <>
      <nav className="navbar">
        <Link to="/" className="navbar-home">
          <strong>{siteConfig.siteName}</strong>
        </Link>

        <ul className="navbar-links">
          {menuPages.map((page) => (
            <li key={page.id}>
              <Link
                to={page.url}
                aria-current={location.pathname === page.url ? "page" : undefined}
              >
                {page.menuTitle || page.title}
              </Link>
            </li>
          ))}
          {menuLinks.map((link) => (
            <li key={link.href}>
              <a href={link.href} target={link.target ?? undefined}>
                {link.text}
              </a>
            </li>
          ))}
          <li>
            <button className="button" onClick={toggleTheme}>
              <span className="icon">◐</span>
            </button>
          </li>
        </ul>
      </nav>

      <main className={bodyClass}>
        <Outlet />
      </main>

      <footer className="footer" />
    </>
  );
}
