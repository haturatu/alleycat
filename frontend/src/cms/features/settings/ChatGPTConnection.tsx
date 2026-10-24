import { useEffect, useRef, useState } from "react";
import { pb } from "@cms/lib/pb";
import { AdminButton, AdminTextField } from "@cms/ui/AriaControls";

type ConnectionStatus = {
  connected: boolean;
  email: string;
  pending: boolean;
  error: string;
};

export default function ChatGPTConnection({ onConnectionChange }: { onConnectionChange: () => void }) {
  const [status, setStatus] = useState<ConnectionStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [pending, setPending] = useState(false);
  const [authorizationUrl, setAuthorizationUrl] = useState("");
  const [callbackUrl, setCallbackUrl] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const controller = useRef<AbortController | null>(null);

  const request = async (path: string, method = "GET", body?: object) => {
    const response = await fetch(`${pb.baseUrl}/api/ai/chatgpt/${path}`, {
      method,
      headers: { Authorization: `Bearer ${pb.authStore.token}`, "Content-Type": "application/json" },
      signal: controller.current?.signal,
      ...(body ? { body: JSON.stringify(body) } : {}),
    });
    const data = response.status === 204 ? {} : await response.json();
    if (!response.ok) throw new Error(data.message || "ChatGPT connection request failed. Try again.");
    return data;
  };

  useEffect(() => {
    const abort = new AbortController();
    controller.current = abort;
    request("status")
      .then((data: ConnectionStatus) => { setStatus(data); setPending(data.pending); })
      .catch((err: Error) => { if (!abort.signal.aborted) setError(err.message); });
    return () => abort.abort();
  }, []);

  useEffect(() => {
    if (!pending) return;
    let active = true;
    let timeout: ReturnType<typeof setTimeout>;
    const poll = async () => {
      try {
        const data: ConnectionStatus = await request("status");
        if (!active) return;
        setStatus(data);
        if (!data.pending) {
          setPending(false);
          setAuthorizationUrl("");
          setCallbackUrl("");
          setError(data.error);
          if (!data.error && data.connected) onConnectionChange();
        } else {
          timeout = setTimeout(poll, 2000);
        }
      } catch (err) {
        if (active) { setPending(false); setError((err as Error).message); }
      }
    };
    timeout = setTimeout(poll, 1000);
    return () => { active = false; clearTimeout(timeout); };
  }, [pending, onConnectionChange]);

  const start = async () => {
    const popup = window.open("about:blank", "_blank");
    if (popup) popup.opener = null;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const data = await request("login", "POST");
      const url = new URL(data.authorization_url);
      if (url.origin !== "https://auth.openai.com") throw new Error("Unexpected ChatGPT sign-in address.");
      setAuthorizationUrl(url.href);
      setPending(true);
      if (popup) popup.location.href = url.href;
    } catch (err) {
      popup?.close();
      if (!controller.current?.signal.aborted) setError((err as Error).message);
    } finally { setBusy(false); }
  };

  const complete = async () => {
    setBusy(true);
    setError("");
    try {
      await request("complete", "POST", { callback_url: callbackUrl });
      const data: ConnectionStatus = await request("status");
      setStatus(data);
      setPending(false);
      setCallbackUrl("");
      setAuthorizationUrl("");
      onConnectionChange();
    } catch (err) { if (!controller.current?.signal.aborted) setError((err as Error).message); }
    finally { setBusy(false); }
  };

  const disconnect = async () => {
    setBusy(true);
    setError("");
    try {
      const data = await request("connection", "DELETE");
      setStatus({ connected: false, email: "", pending: false, error: "" });
      setPending(false);
      setAuthorizationUrl("");
      setCallbackUrl("");
      if (!data.revoked) setNotice("Disconnected here. Remote sign-out could not be confirmed; remove alleycat in ChatGPT Settings to end its access.");
      onConnectionChange();
    } catch (err) { if (!controller.current?.signal.aborted) setError((err as Error).message); }
    finally { setBusy(false); }
  };

  const cancel = async () => {
    setBusy(true);
    try {
      await request("login", "DELETE");
      setPending(false);
      setAuthorizationUrl("");
      setCallbackUrl("");
      setError("");
    } catch (err) { if (!controller.current?.signal.aborted) setError((err as Error).message); }
    finally { setBusy(false); }
  };

  return (
    <div className="admin-settings-subsection">
      <h3>ChatGPT connection</h3>
      <p className="admin-note">Use your ChatGPT plan for this site’s slug generation and translation. Your plan’s shared limits apply.</p>
      <p role="status">{status ? status.connected ? `Connected${status.email ? ` as ${status.email}` : ""}` : "Not connected" : "Checking connection…"}</p>
      {error ? <p role="alert" className="admin-error">{error}</p> : null}
      {notice ? <p role="status" className="admin-note">{notice}</p> : null}
      {pending ? (
        <>
          <p className="admin-note">Finish sign-in in the ChatGPT tab, then return here. Sign-in expires after 10 minutes.</p>
          {authorizationUrl ? <a href={authorizationUrl} target="_blank" rel="noreferrer">Open ChatGPT sign-in</a> : null}
          <p className="admin-note">If the completion page cannot open, copy its full address from the browser and paste it below.</p>
          <AdminTextField label="Completion address" type="url" value={callbackUrl} onChange={setCallbackUrl} />
          <AdminButton className="admin-primary" disabled={busy || !callbackUrl.trim()} onPress={complete}>Complete connection</AdminButton>
          <AdminButton className="admin-ghost" disabled={busy} onPress={cancel}>Cancel sign-in</AdminButton>
        </>
      ) : status?.connected ? (
        <AdminButton className="admin-ghost" disabled={busy} onPress={disconnect}>{busy ? "Disconnecting…" : "Disconnect ChatGPT"}</AdminButton>
      ) : (
        <AdminButton className="admin-primary" disabled={busy || !status} onPress={start}>{busy ? "Preparing sign-in…" : "Continue with ChatGPT"}</AdminButton>
      )}
      <p className="admin-note"><a href="https://chatgpt.com/settings/usage" target="_blank" rel="noreferrer">Manage ChatGPT app usage and permissions</a></p>
    </div>
  );
}
