import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import { useEffect, useRef } from "react";
import { pb } from "../lib/pb";

export default function RichEditor({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  const lastValueRef = useRef(value);
  const rewriteSeqRef = useRef(0);
  const mediaCaptionCache = useRef<Map<string, string>>(new Map());

  const rewriteMediaUrls = async (html: string) => {
    const re = /(?:https?:\/\/[^"'\\s)]+)?\/api\/files\/([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_-]+)\/([^"'\\s)]+)/g;
    const matches = [...html.matchAll(re)];
    if (matches.length === 0) return html;

    await Promise.all(
      matches.map(async (match) => {
        const collection = match[1];
        const recordId = match[2];
        if (collection !== "media" && collection !== "pbc_2708086759") return;
        if (mediaCaptionCache.current.has(recordId)) return;
        try {
          const record = await pb.collection("media").getOne(recordId);
          const mediaPath = typeof record.path === "string" ? record.path.trim() : "";
          const fallback = typeof record.caption === "string" ? record.caption.trim() : "";
          if (mediaPath) {
            mediaCaptionCache.current.set(recordId, mediaPath);
          } else if (fallback) {
            mediaCaptionCache.current.set(recordId, fallback);
          }
        } catch {
          // ignore
        }
      })
    );

    return html.replace(re, (full, collection, recordId, filename) => {
      const mediaPath = mediaCaptionCache.current.get(recordId);
      if (!mediaPath) {
        return `/api/files/${collection}/${recordId}/${filename}`;
      }
      if (mediaPath.startsWith("http://") || mediaPath.startsWith("https://")) return mediaPath;
      return mediaPath.startsWith("/") ? mediaPath : `/${mediaPath}`;
    });
  };

  const uploadAndInsertImage = async (file: File, editor: any) => {
    if (!editor) return;
    const form = new FormData();
    form.set("file", file);
    form.set("public", "true");
    form.set("alt", file.name);
    try {
      const record = await pb.collection("media").create(form);
      const filename = record.file as string;
      let mediaPath = typeof record.path === "string" ? record.path.trim() : "";
      if (!mediaPath && filename) {
        const normalized = normalizeUploadFilename(filename);
        mediaPath = `/uploads/${normalized}`;
        try {
          await pb.collection("media").update(record.id, { path: mediaPath });
        } catch {
          // ignore path update failure
        }
      }
      if (mediaPath) {
        mediaCaptionCache.current.set(record.id, mediaPath);
      }
      const url = mediaPath || pb.files.getURL(record, filename);
      editor.chain().focus().setImage({ src: url, alt: file.name }).run();
    } catch (err) {
      console.error(err);
      alert("画像のアップロードに失敗しました。");
    }
  };
  const editor = useEditor({
    extensions: [
      StarterKit,
      Image.configure({
        inline: false,
        allowBase64: false,
      }),
    ],
    content: value,
    onUpdate: ({ editor }) => {
      const html = editor.getHTML();
      if (html !== lastValueRef.current) {
        lastValueRef.current = html;
        onChange(html);
      }
    },
    editorProps: {
      handlePaste: (_view, event) => {
        const items = Array.from(event.clipboardData?.items || []);
        const files = items
          .map((item) => (item.kind === "file" ? item.getAsFile() : null))
          .filter((file): file is File => Boolean(file) && file!.type.startsWith("image/"));

        if (files.length === 0) {
          return false;
        }

        event.preventDefault();
        files.forEach((file) => void uploadAndInsertImage(file, editor));
        return true;
      },
      handleDrop: (_view, event) => {
        const files = Array.from(event.dataTransfer?.files || []).filter((file) =>
          file.type.startsWith("image/")
        );
        if (files.length === 0) {
          return false;
        }
        event.preventDefault();
        files.forEach((file) => void uploadAndInsertImage(file, editor));
        return true;
      },
    },
  });

  if (!editor) return null;

  useEffect(() => {
    if (!editor) return;
    const hasFocus = editor.view?.hasFocus?.() ?? false;
    if (value !== lastValueRef.current && !hasFocus) {
      const seq = ++rewriteSeqRef.current;
      (async () => {
        const rewritten = await rewriteMediaUrls(value || "<p></p>");
        if (seq !== rewriteSeqRef.current) return;
        editor.commands.setContent(rewritten || "<p></p>", false);
        lastValueRef.current = rewritten;
      })();
    }
  }, [editor, value]);

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

  return (
    <div className="editor">
      <div className="editor-toolbar">
        <button onClick={() => editor.chain().focus().toggleBold().run()} type="button">
          Bold
        </button>
        <button onClick={() => editor.chain().focus().toggleItalic().run()} type="button">
          Italic
        </button>
        <button onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()} type="button">
          H2
        </button>
        <button onClick={() => editor.chain().focus().toggleBulletList().run()} type="button">
          Bullet
        </button>
        <button onClick={() => editor.chain().focus().toggleCode().run()} type="button">
          Code
        </button>
        <button onClick={() => editor.chain().focus().toggleCodeBlock().run()} type="button">
          Code Block
        </button>
      </div>
      <EditorContent editor={editor} />
    </div>
  );
}
