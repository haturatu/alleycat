import { Navigate } from "react-router-dom";
import { hasRole, isAuthed } from "../lib/pb";

export default function RequireAdmin({ children }: { children: React.ReactNode }) {
  if (!isAuthed() || !hasRole(["admin", "editor"])) {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}
