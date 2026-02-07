import { useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { hasRole, isAuthed, pb } from "../lib/pb";
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
      setError("ログインに失敗しました。");
    }
  };

  return (
    <div className="admin-login">
      <form onSubmit={submit} className="admin-card">
        <h1>Admin Login</h1>
        <label>
          Email
          <input value={email} onChange={(e) => setEmail(e.target.value)} type="email" required />
        </label>
        <label>
          Password
          <input value={password} onChange={(e) => setPassword(e.target.value)} type="password" required />
        </label>
        {error && <p className="admin-error">{error}</p>}
        <button className="admin-primary" type="submit">
          Login
        </button>
      </form>
    </div>
  );
}
