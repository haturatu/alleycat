import { useEffect } from "react";

export default function useUnsavedChangesGuard(
  isDirty: boolean,
  message = "You have unsaved changes. Leave without saving?"
) {
  useEffect(() => {
    if (!isDirty) return;
    const beforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", beforeUnload);
    return () => window.removeEventListener("beforeunload", beforeUnload);
  }, [isDirty]);

  useEffect(() => {
    if (!isDirty) return;
    const onClickCapture = (event: MouseEvent) => {
      const target = event.target as HTMLElement | null;
      const anchor = target?.closest?.("a[href]") as HTMLAnchorElement | null;
      if (!anchor) return;
      if (anchor.target === "_blank" || anchor.hasAttribute("download")) return;
      const href = anchor.getAttribute("href");
      if (!href || href.startsWith("#")) return;
      const next = new URL(anchor.href, window.location.href);
      if (next.origin !== window.location.origin) return;
      const currentPath = `${window.location.pathname}${window.location.search}${window.location.hash}`;
      const nextPath = `${next.pathname}${next.search}${next.hash}`;
      if (currentPath === nextPath) return;
      if (window.confirm(message)) return;
      event.preventDefault();
      event.stopPropagation();
    };
    document.addEventListener("click", onClickCapture, true);
    return () => document.removeEventListener("click", onClickCapture, true);
  }, [isDirty, message]);
}
