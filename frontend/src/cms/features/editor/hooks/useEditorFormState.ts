import { useState } from "react";

export default function useEditorFormState() {
  const [saveMessage, setSaveMessage] = useState("");
  const [isDirty, setIsDirty] = useState(false);
  const [lastSavedAt, setLastSavedAt] = useState<string>("");

  const clearSaveMessage = () => setSaveMessage("");
  const markDirty = () => {
    setSaveMessage("");
    setIsDirty(true);
  };
  const markSaved = (message = "") => {
    setIsDirty(false);
    setSaveMessage(message);
    setLastSavedAt(new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }));
  };

  return {
    saveMessage,
    setSaveMessage,
    clearSaveMessage,
    isDirty,
    setIsDirty,
    lastSavedAt,
    markDirty,
    markSaved,
  };
}
