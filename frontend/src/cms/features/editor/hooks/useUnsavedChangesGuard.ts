import { useEffect } from "react";
import { useBeforeUnload, useBlocker } from "react-router-dom";

export default function useUnsavedChangesGuard(
  isDirty: boolean,
  message = "You have unsaved changes. Leave without saving?"
) {
  useBeforeUnload(
    (event) => {
      if (!isDirty) return;
      event.preventDefault();
      event.returnValue = "";
    },
    { capture: true }
  );

  const blocker = useBlocker(isDirty);

  useEffect(() => {
    if (blocker.state !== "blocked") return;
    if (window.confirm(message)) {
      blocker.proceed();
      return;
    }
    blocker.reset();
  }, [blocker, message]);
}
