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

export const uploadImageAndGetURL = async (file: File): Promise<string> => {
  const form = new FormData();
  form.set("file", file);
  form.set("public", "true");
  form.set("alt", file.name);

  const record = await pb.collection("media").create(form);
  const filename = String(record.file || "");
  let mediaPath = typeof record.path === "string" ? record.path.trim() : "";
  if (!mediaPath && filename) {
    const normalized = normalizeUploadFilename(filename);
    mediaPath = `/uploads/${normalized}`;
    try {
      await pb.collection("media").update(record.id, { path: mediaPath });
    } catch {
      // Ignore path update failures, fallback URL still works.
    }
  }

  if (mediaPath) {
    if (mediaPath.startsWith("http://") || mediaPath.startsWith("https://")) return mediaPath;
    return mediaPath.startsWith("/") ? mediaPath : `/${mediaPath}`;
  }
  return pb.files.getURL(record, filename);
};
