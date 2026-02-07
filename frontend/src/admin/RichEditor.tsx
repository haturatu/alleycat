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
  const uploadAndInsertImage = async (file: File, editor: any) => {
    if (!editor) return;
    const form = new FormData();
    form.set("file", file);
    form.set("public", "true");
    form.set("alt", file.name);
    try {
      const record = await pb.collection("media").create(form);
      const filename = record.file as string;
      const url = pb.files.getURL(record, filename);
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
      editor.commands.setContent(value || "<p></p>", false);
      lastValueRef.current = value;
    }
  }, [editor, value]);

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
      </div>
      <EditorContent editor={editor} />
    </div>
  );
}
