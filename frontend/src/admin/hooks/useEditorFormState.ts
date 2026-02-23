import { useState } from "react";

export default function useEditorFormState() {
  const [saveMessage, setSaveMessage] = useState("");
  const [isDirty, setIsDirty] = useState(false);

  const clearSaveMessage = () => setSaveMessage("");
  const markDirty = () => {
    setSaveMessage("");
    setIsDirty(true);
  };
  const markSaved = (message = "") => {
    setIsDirty(false);
    setSaveMessage(message);
  };

  return {
    saveMessage,
    setSaveMessage,
    clearSaveMessage,
    isDirty,
    setIsDirty,
    markDirty,
    markSaved,
  };
}
