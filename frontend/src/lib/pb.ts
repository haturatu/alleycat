import PocketBase from "pocketbase";

const inferredUrl =
  typeof window !== "undefined"
    ? `http://${window.location.hostname}:8090`
    : "http://127.0.0.1:8090";

const envUrl = import.meta.env.VITE_PB_URL;
const normalizedEnvUrl =
  envUrl && envUrl.includes("0.0.0.0")
    ? envUrl.replace(
        "0.0.0.0",
        typeof window !== "undefined" ? window.location.hostname : "127.0.0.1"
      )
    : envUrl;

const baseUrl =
  normalizedEnvUrl && normalizedEnvUrl.startsWith("/")
    ? typeof window !== "undefined"
      ? `${window.location.origin}${normalizedEnvUrl}`
      : `http://127.0.0.1:8090${normalizedEnvUrl}`
    : normalizedEnvUrl || (typeof window !== "undefined" ? window.location.origin : inferredUrl);

export const pb = new PocketBase(baseUrl);
pb.autoCancellation(false);

export type PostRecord = {
  id: string;
  title: string;
  slug: string;
  body: string;
  content?: string;
  excerpt?: string;
  tags?: string;
  category?: string;
  author?: string;
  featured_image?: string;
  attachments?: string[];
  published_at?: string;
  published?: boolean;
};

export type PostTranslationRecord = {
  id: string;
  source_post: string;
  locale: string;
  title: string;
  slug: string;
  body: string;
  excerpt?: string;
  tags?: string;
  category?: string;
  author?: string;
  featured_image?: string;
  attachments?: string[];
  published_at?: string;
  published?: boolean;
  translation_done?: boolean;
};

export type PageRecord = {
  id: string;
  title: string;
  slug: string;
  url: string;
  body: string;
  menuVisible?: boolean;
  menuOrder?: number;
  menuTitle?: string;
  published_at?: string;
  published?: boolean;
};

export const isAuthed = () => pb.authStore.isValid;

export const hasRole = (roles: string[]) => {
  const role = (pb.authStore.model as { role?: string } | null)?.role;
  return role ? roles.includes(role) : false;
};

export const fileUrl = (collection: string, recordId: string, filename?: string) => {
  if (!filename) return "";
  return `${pb.baseUrl}/api/files/${collection}/${recordId}/${filename}`;
};
