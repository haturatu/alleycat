import { useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { hasRole, isAuthed, pb } from "../lib/pb";
import { AdminButton, AdminTextField } from "./components/AriaControls";
import useAdminPageTitle from "./hooks/useAdminPageTitle";
import "./styles/index.css";

export default function AdminLogin() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

  useAdminPageTitle("Login");

  if (isAuthed() && hasRole(["admin", "editor"])) {
    return <Navigate to="/posts" replace />;
  }
  if (isAuthed() && !hasRole(["admin", "editor"])) {
    pb.authStore.clear();
  }

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError("");
    try {
      await pb.collection("cms_users").authWithPassword(email, password);
      navigate("/posts");
    } catch (err) {
      setError("Sign-in failed. Check your email address and password.");
    }
  };

  return (
    <div className="admin-login">
      <div className="admin-login-shell">
        <section className="admin-login-panel admin-login-story">
          <p className="admin-eyebrow">Editorial Console</p>
          <h1>Shape the next publish cycle with intention.</h1>
          <p className="admin-login-copy">
            Review drafts, manage structure, and push updates through a workspace designed for deliberate publishing.
          </p>
          <p className="admin-login-kicker">Draft clearly. Review deliberately. Publish without losing the thread.</p>
          <div className="admin-login-metrics">
            <article className="admin-summary-card">
              <span className="admin-summary-label">Workspace</span>
              <strong>Calm</strong>
              <p>Focused editing surfaces with operational controls close at hand.</p>
            </article>
            <article className="admin-summary-card">
              <span className="admin-summary-label">Access</span>
              <strong>Role-based</strong>
              <p>Admins and editors enter the same publishing room with clear boundaries.</p>
            </article>
          </div>
        </section>
        <form onSubmit={submit} className="admin-card admin-login-panel admin-login-form">
          <div className="admin-login-head">
            <p className="admin-section-label">Secure Access</p>
            <h2>Admin Sign In</h2>
            <p className="admin-note">Use your CMS credentials to enter the editorial workspace.</p>
          </div>
          <AdminTextField label="Email" value={email} onChange={setEmail} type="email" required />
          <AdminTextField
            label="Password"
            value={password}
            onChange={setPassword}
            type="password"
            required
          />
          {error && <p className="admin-error">{error}</p>}
          <AdminButton className="admin-primary" type="submit">
            Sign In
          </AdminButton>
        </form>
      </div>
    </div>
  );
}
