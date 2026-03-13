import { useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { hasRole, isAuthed, pb } from "../lib/pb";
import { AdminButton, AdminTextField } from "./components/AriaControls";
import "../styles/admin.css";

export default function AdminLogin() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

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
      setError("Login failed.");
    }
  };

  return (
    <div className="admin-login">
      <form onSubmit={submit} className="admin-card">
        <h1>Admin Login</h1>
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
          Login
        </AdminButton>
      </form>
    </div>
  );
}
