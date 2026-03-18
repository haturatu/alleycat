import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useState } from "react";
import { pb } from "../lib/pb";
import { AdminButton, AdminDialog } from "./components/AriaControls";
import "./styles/index.css";
import "highlight.js/styles/github-dark.css";

export default function AdminLayout() {
  const navigate = useNavigate();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const logout = () => {
    pb.authStore.clear();
    navigate("/");
  };

  const closeSidebar = () => setSidebarOpen(false);

  const sidebarContent = (
    <>
        <div className="admin-brand">
          <p className="admin-eyebrow">Admin</p>
          <h2>Admin</h2>
          <p className="admin-brand-note">Manage content, structure, and settings from one workspace.</p>
        </div>
        <nav aria-label="Admin" className="admin-nav">
        <NavLink className={({ isActive }) => (isActive ? "is-current" : undefined)} to="/posts" onClick={closeSidebar}>
          Posts
        </NavLink>
        <NavLink className={({ isActive }) => (isActive ? "is-current" : undefined)} to="/pages" onClick={closeSidebar}>
          Pages
        </NavLink>
        <NavLink className={({ isActive }) => (isActive ? "is-current" : undefined)} to="/settings" onClick={closeSidebar}>
          Settings
        </NavLink>
        <AdminButton className="admin-ghost" onPress={() => { closeSidebar(); logout(); }}>
          Sign Out
        </AdminButton>
      </nav>
    </>
  );

  return (
    <div className="admin-shell" data-sidebar-open={sidebarOpen}>
      <a className="admin-skip-link" href="#admin-content">
        Skip to content
      </a>
      <aside className="admin-sidebar">{sidebarContent}</aside>
      <div className="admin-sidebar-mobile">
        <AdminDialog
          open={sidebarOpen}
          onClose={closeSidebar}
          title="Admin navigation"
          overlayClassName="admin-drawer-backdrop"
          shellClassName="admin-drawer-shell"
        >
          <div className="admin-modal-head">
            <div>
              <p className="admin-section-label">Admin</p>
              <h2>Admin</h2>
            </div>
            <AdminButton ariaLabel="Close navigation" className="admin-modal-close admin-icon-button" onPress={closeSidebar}>
              ←
            </AdminButton>
          </div>
          <div className="admin-modal-body admin-mobile-nav">{sidebarContent}</div>
        </AdminDialog>
      </div>
      <main className="admin-main" id="admin-content" tabIndex={-1}>
        <div className="admin-main-chrome">
          <p className="admin-eyebrow">Publishing</p>
          <p className="admin-main-note">Structured tools for drafting, reviewing, and shipping content.</p>
        </div>
        <AdminButton
          ariaLabel="Toggle menu"
          className="admin-sidebar-toggle"
          onPress={() => setSidebarOpen((open) => !open)}
          type="button"
        >
          ☰
        </AdminButton>
        <Outlet />
      </main>
    </div>
  );
}
