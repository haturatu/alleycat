import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import Link from "@tiptap/extension-link";
import { useEffect, useRef } from "react";
import { pb } from "../lib/pb";
import { uploadImageAndGetURL } from "./mediaUpload";
import { AdminButton } from "./components/AriaControls";

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
    try {
      const url = await uploadImageAndGetURL(file);
      editor.chain().focus().setImage({ src: url, alt: file.name }).run();
    } catch (err) {
      console.error(err);
      alert("Failed to upload image.");
    }
  };
  const editor = useEditor({
    extensions: [
      StarterKit,
      Image.configure({
        inline: false,
        allowBase64: false,
      }),
      Link.configure({
        openOnClick: false,
        autolink: true,
        linkOnPaste: true,
        protocols: ["http", "https"],
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

  const setLink = () => {
    const previousUrl = editor.getAttributes("link").href as string | undefined;
    const url = window.prompt("URL", previousUrl ?? "https://");
    if (url === null) return;

    const trimmed = url.trim();
    if (!trimmed) {
      editor.chain().focus().extendMarkRange("link").unsetLink().run();
      return;
    }

    const normalized = /^https?:\/\//i.test(trimmed) ? trimmed : `https://${trimmed}`;
    editor.chain().focus().extendMarkRange("link").setLink({ href: normalized }).run();
  };

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
  return (
    <div className="editor">
      <div aria-label="Rich text formatting" className="editor-toolbar" role="toolbar">
        <AdminButton
          ariaLabel="Toggle bold"
          ariaPressed={editor.isActive("bold")}
          onPress={() => editor.chain().focus().toggleBold().run()}
          type="button"
        >
          Bold
        </AdminButton>
        <AdminButton
          ariaLabel="Toggle italic"
          ariaPressed={editor.isActive("italic")}
          onPress={() => editor.chain().focus().toggleItalic().run()}
          type="button"
        >
          Italic
        </AdminButton>
        <AdminButton
          ariaLabel="Toggle heading level 2"
          ariaPressed={editor.isActive("heading", { level: 2 })}
          onPress={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
          type="button"
        >
          H2
        </AdminButton>
        <AdminButton
          ariaLabel="Toggle bullet list"
          ariaPressed={editor.isActive("bulletList")}
          onPress={() => editor.chain().focus().toggleBulletList().run()}
          type="button"
        >
          Bullet
        </AdminButton>
        <AdminButton
          ariaLabel="Toggle inline code"
          ariaPressed={editor.isActive("code")}
          onPress={() => editor.chain().focus().toggleCode().run()}
          type="button"
        >
          Code
        </AdminButton>
        <AdminButton
          ariaLabel="Toggle code block"
          ariaPressed={editor.isActive("codeBlock")}
          onPress={() => editor.chain().focus().toggleCodeBlock().run()}
          type="button"
        >
          Code Block
        </AdminButton>
        <AdminButton ariaLabel="Set link" ariaPressed={editor.isActive("link")} onPress={setLink} type="button">
          Link
        </AdminButton>
        <AdminButton ariaLabel="Remove link" onPress={() => editor.chain().focus().unsetLink().run()} type="button">
          Unlink
        </AdminButton>
      </div>
      <EditorContent editor={editor} />
    </div>
  );
}
