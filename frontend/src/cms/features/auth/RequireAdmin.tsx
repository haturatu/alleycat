import { useEffect, useState } from "react";
import { Navigate } from "react-router-dom";
import { hasRole, isAuthed, pb } from "@cms/lib/pb";

export default function RequireAdmin({ children }: { children: React.ReactNode }) {
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    let active = true;
    const validateSession = async () => {
      if (!isAuthed()) {
        if (active) setChecking(false);
        return;
      }
      try {
        await pb.collection("cms_users").authRefresh();
        if (!hasRole(["admin", "editor"])) pb.authStore.clear();
      } catch {
        pb.authStore.clear();
      } finally {
        if (active) setChecking(false);
      }
    };
    void validateSession();
    return () => {
      active = false;
    };
  }, []);

  if (checking) return null;
  if (!isAuthed() || !hasRole(["admin", "editor"])) {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}
