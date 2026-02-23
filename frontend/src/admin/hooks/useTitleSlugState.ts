import { slugify } from "../../utils/text";
import { validateSlug, validateTitle } from "../validation";

type UseTitleSlugStateParams = {
  title: string;
  slugEditedManually: boolean;
  setTitle: (value: string) => void;
  setSlug: (value: string) => void;
  setSlugEditedManually: (value: boolean) => void;
  markDirty: () => void;
  setFieldError: (field: "title" | "slug", message?: string) => void;
};

export default function useTitleSlugState({
  title,
  slugEditedManually,
  setTitle,
  setSlug,
  setSlugEditedManually,
  markDirty,
  setFieldError,
}: UseTitleSlugStateParams) {
  const onTitleChange = (next: string) => {
    setTitle(next);
    markDirty();
    setFieldError("title", validateTitle(next));
    if (!slugEditedManually) {
      const nextSlug = slugify(next);
      setSlug(nextSlug);
      setFieldError("slug", validateSlug(nextSlug));
    }
  };

  const onSlugChange = (next: string) => {
    setSlug(next);
    setSlugEditedManually(true);
    markDirty();
    setFieldError("slug", validateSlug(next));
  };

  const onAutoSlug = () => {
    const auto = slugify(title);
    setSlug(auto);
    setSlugEditedManually(false);
    markDirty();
    setFieldError("slug", validateSlug(auto));
  };

  return {
    onTitleChange,
    onSlugChange,
    onAutoSlug,
  };
}
