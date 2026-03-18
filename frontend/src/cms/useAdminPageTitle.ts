import { useEffect } from "react";

const ADMIN_TITLE_SUFFIX = "Admin";

export default function useAdminPageTitle(title: string) {
  useEffect(() => {
    document.title = `${title} | ${ADMIN_TITLE_SUFFIX}`;
  }, [title]);
}
