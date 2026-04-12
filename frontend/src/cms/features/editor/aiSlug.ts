import { pb } from "@cms/lib/pb";

type AISlugStatusResponse = {
  enabled?: boolean;
};

type AISlugResponse = {
  slug?: string;
};

const buildAuthHeaders = () => {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  const token = pb.authStore.token;
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
};

export const fetchAISlugStatus = async () => {
  const response = await fetch(`${pb.baseUrl}/api/ai/slug/status`, {
    headers: buildAuthHeaders(),
  });
  if (!response.ok) {
    throw new Error("Failed to load AI slug status.");
  }
  const data = (await response.json()) as AISlugStatusResponse;
  return Boolean(data.enabled);
};

export const generateAISlug = async (title: string) => {
  const response = await fetch(`${pb.baseUrl}/api/ai/slug`, {
    method: "POST",
    headers: buildAuthHeaders(),
    body: JSON.stringify({ title }),
  });

  const data = (await response.json().catch(() => ({}))) as AISlugResponse & { message?: string };
  if (!response.ok) {
    throw new Error(data.message || "Failed to generate AI slug.");
  }

  const slug = String(data.slug || "").trim();
  if (!slug) {
    throw new Error("AI slug generation returned an empty slug.");
  }

  return slug;
};
