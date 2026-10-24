import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import ChatGPTConnection from "./ChatGPTConnection";

vi.mock("@cms/lib/pb", () => ({ pb: { baseUrl: "https://cms.example", authStore: { token: "cms-token" } } }));
vi.mock("@cms/ui/AriaControls", () => ({
  AdminButton: ({ children, onPress, disabled }: { children: React.ReactNode; onPress: () => void; disabled?: boolean }) => <button disabled={disabled} onClick={onPress}>{children}</button>,
  AdminTextField: ({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) => <label>{label}<input value={value} onInput={(e) => onChange(e.currentTarget.value)} /></label>,
}));

let container: HTMLDivElement;
let root: Root;
let fetchMock: ReturnType<typeof vi.fn>;
const disconnected = { connected: false, pending: false, email: "", error: "" };
const connected = { connected: true, pending: false, email: "writer@example.com", error: "" };
const jsonResponse = (body: object, status = 200) => ({ ok: status >= 200 && status < 300, status, json: async () => body });

beforeEach(() => {
  (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

const button = (label: string) => [...container.querySelectorAll("button")].find((item) => item.textContent === label)!;

test("starts sign-in and completes a remote callback without browser token storage", async () => {
  const changed = vi.fn();
  const popup = { opener: {}, location: { href: "about:blank" }, close: vi.fn() };
  vi.spyOn(window, "open").mockReturnValue(popup as unknown as Window);
  fetchMock
    .mockResolvedValueOnce(jsonResponse(disconnected))
    .mockResolvedValueOnce(jsonResponse({ authorization_url: "https://auth.openai.com/api/accounts/authorize?state=test" }))
    .mockResolvedValueOnce(jsonResponse({ connected: true }))
    .mockResolvedValueOnce(jsonResponse(connected));
  await act(async () => root.render(<ChatGPTConnection onConnectionChange={changed} />));
  await act(async () => button("Continue with ChatGPT").click());
  expect(popup.location.href).toContain("https://auth.openai.com/");
  expect(popup.opener).toBeNull();
  expect(container.textContent).toContain("Completion address");
  const callback = "http://127.0.0.1:1455/auth/callback?state=test&code=code";
  const input = container.querySelector("input")!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, callback);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () => button("Complete connection").click());
  expect(fetchMock.mock.calls[2][1].body).toBe(JSON.stringify({ callback_url: callback }));
  expect(fetchMock.mock.calls[2][1].headers.Authorization).toBe("Bearer cms-token");
  expect(container.textContent).toContain("Connected as writer@example.com");
  expect(changed).toHaveBeenCalledOnce();
  expect(container.querySelector("input")).toBeNull();
  expect(localStorage.length).toBe(0);
});

test("polls for loopback completion and refreshes model choices", async () => {
  vi.useFakeTimers();
  const changed = vi.fn();
  vi.spyOn(window, "open").mockReturnValue(null);
  fetchMock
    .mockResolvedValueOnce(jsonResponse(disconnected))
    .mockResolvedValueOnce(jsonResponse({ authorization_url: "https://auth.openai.com/api/accounts/authorize?state=test" }))
    .mockResolvedValueOnce(jsonResponse(connected));
  await act(async () => root.render(<ChatGPTConnection onConnectionChange={changed} />));
  await act(async () => button("Continue with ChatGPT").click());
  expect(container.querySelector('a[href^="https://auth.openai.com/"]')).not.toBeNull();
  await act(async () => { await vi.advanceTimersByTimeAsync(1000); });
  expect(container.textContent).toContain("Connected as writer@example.com");
  expect(changed).toHaveBeenCalledOnce();
  expect(button("Disconnect ChatGPT")).toBeDefined();
});

test("rejects an unexpected sign-in host", async () => {
  const popup = { opener: {}, location: { href: "about:blank" }, close: vi.fn() };
  vi.spyOn(window, "open").mockReturnValue(popup as unknown as Window);
  fetchMock
    .mockResolvedValueOnce(jsonResponse(disconnected))
    .mockResolvedValueOnce(jsonResponse({ authorization_url: "https://evil.example/authorize" }));
  await act(async () => root.render(<ChatGPTConnection onConnectionChange={vi.fn()} />));
  await act(async () => button("Continue with ChatGPT").click());
  expect(popup.close).toHaveBeenCalledOnce();
  expect(popup.location.href).toBe("about:blank");
  expect(container.querySelector('[role="alert"]')?.textContent).toContain("Unexpected ChatGPT sign-in address");
});

test("explains remote revocation failures after local disconnect", async () => {
  const changed = vi.fn();
  fetchMock.mockResolvedValueOnce(jsonResponse(connected)).mockResolvedValueOnce(jsonResponse({ connected: false, revoked: false }));
  await act(async () => root.render(<ChatGPTConnection onConnectionChange={changed} />));
  await act(async () => button("Disconnect ChatGPT").click());
  expect(container.textContent).toContain("Remote sign-out could not be confirmed");
  expect(container.textContent).toContain("Not connected");
  expect(changed).toHaveBeenCalledOnce();
});
