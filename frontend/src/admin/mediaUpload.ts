import { ClientResponseError } from "pocketbase";
import { sha256 } from "js-sha256";
import { pb } from "../lib/pb";

const normalizeUploadFilename = (filename: string) => {
  const dot = filename.lastIndexOf(".");
  if (dot <= 0) return filename;
  const base = filename.slice(0, dot);
  const ext = filename.slice(dot);
  const underscore = base.lastIndexOf("_");
  if (underscore === -1) return filename;
  const suffix = base.slice(underscore + 1);
  if (suffix.length < 6 || suffix.length > 16) return filename;
  if (!/^[a-zA-Z0-9]+$/.test(suffix)) return filename;
  return `${base.slice(0, underscore)}${ext}`;
};

const buildUploadPath = (filename: string, checksum: string) => {
  const normalizedChecksum = checksum.trim().toLowerCase();
  if (normalizedChecksum) {
    const dot = filename.lastIndexOf(".");
    const ext = dot > 0 ? filename.slice(dot).toLowerCase() : "";
    return `/uploads/${normalizedChecksum}${ext}`;
  }
  const normalized = normalizeUploadFilename(filename);
  return `/uploads/${normalized}`;
};

const escapeFilterString = (value: string) => value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');

const hashFileSHA256 = async (file: File) => {
  const data = new Uint8Array(await file.arrayBuffer());
  return sha256(data);
};

type MediaRecord = {
  id: string;
  file?: string;
  path?: string;
  caption?: string;
  checksum?: string;
};

const resolveMediaURL = (record: MediaRecord) => {
  const filename = String(record.file || "");
  const checksum = typeof record.checksum === "string" ? record.checksum.trim() : "";
  let mediaPath = typeof record.path === "string" ? record.path.trim() : "";
  if (!mediaPath && filename) {
    mediaPath = buildUploadPath(filename, checksum);
  }
  if (mediaPath) {
    if (mediaPath.startsWith("http://") || mediaPath.startsWith("https://")) return mediaPath;
    return mediaPath.startsWith("/") ? mediaPath : `/${mediaPath}`;
  }
  return pb.files.getURL(record, filename);
};

const findMediaByChecksum = async (checksum: string): Promise<MediaRecord | null> => {
  try {
    return await pb
      .collection("media")
      .getFirstListItem<MediaRecord>(`checksum = "${escapeFilterString(checksum)}"`);
  } catch (error) {
    if (error instanceof ClientResponseError && error.status === 404) {
      return null;
    }
    throw error;
  }
};

export const uploadImageAndGetURL = async (file: File): Promise<string> => {
  const checksum = await hashFileSHA256(file);
  const existing = await findMediaByChecksum(checksum);
  if (existing) return resolveMediaURL(existing);

  const form = new FormData();
  form.set("file", file);
  form.set("public", "true");
  form.set("alt", file.name);
  form.set("checksum", checksum);

  let record: MediaRecord;
  try {
    record = await pb.collection("media").create<MediaRecord>(form);
  } catch (error) {
    if (
      error instanceof ClientResponseError &&
      error.status === 400 &&
      error.response?.data &&
      typeof error.response.data === "object" &&
      "checksum" in error.response.data
    ) {
      const alreadyExists = await findMediaByChecksum(checksum);
      if (alreadyExists) return resolveMediaURL(alreadyExists);
    }
    throw error;
  }

  const filename = String(record.file || "");
  const mediaPath = typeof record.path === "string" ? record.path.trim() : "";
  if (!mediaPath && filename) {
    const nextPath = buildUploadPath(filename, checksum);
    try {
      await pb.collection("media").update(record.id, { path: nextPath });
      record.path = nextPath;
    } catch {
      // Ignore path update failures, fallback URL still works.
    }
  }

  return resolveMediaURL(record);
};
