import { http } from "@/utils/http";
import { getToken } from "@/utils/auth";
import type { GoUser } from "./user";

export type ChatUser = GoUser & {
  online: boolean;
  unread_count: number;
};

export type ChatMessageType =
  | "text"
  | "emoji"
  | "image"
  | "video"
  | "audio"
  | "file";

export type ChatMessage = {
  id: number;
  from_user_id: number;
  to_user_id: number;
  message_type: ChatMessageType;
  content: string;
  media_url: string;
  file_name: string;
  mime_type: string;
  file_size: number;
  transcript: string;
  translation: string;
  is_read: boolean;
  created_at: string;
};

export type ChatUploadResult = {
  url: string;
  file_name: string;
  mime_type: string;
  file_size: number;
  message_type: Exclude<ChatMessageType, "text" | "emoji">;
};

export type ChatTranslateResult = {
  translated_text: string;
  target_lang: string;
};

export const getChatUsers = () => {
  return http.request<{ users: ChatUser[] }>("get", "/api/v1/chat/users");
};

export const getChatHistory = (userId: number) => {
  return http.request<{ messages: ChatMessage[] }>(
    "get",
    `/api/v1/chat/history/${userId}`
  );
};

export const uploadChatFile = (file: File) => {
  const formData = new FormData();
  formData.append("file", file);
  return http.request<ChatUploadResult>("post", "/api/v1/chat/upload", {
    data: formData,
    headers: {
      "Content-Type": "multipart/form-data"
    }
  });
};

export const markChatRead = (fromUserId: number) => {
  return http.request("post", `/api/v1/chat/read/${fromUserId}`);
};

export const translateChatText = (text: string, targetLang = "zh") => {
  return http.request<ChatTranslateResult>("post", "/api/v1/chat/translate", {
    data: {
      text,
      target_lang: targetLang
    }
  });
};

export const createChatSocket = () => {
  const token = getToken()?.accessToken;
  if (!token) return null;

  const apiBase = import.meta.env.VITE_API_BASE_URL || "";
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const base = apiBase
    ? apiBase.replace(/^http/, "ws")
    : `${protocol}://${window.location.host}`;

  return new WebSocket(
    `${base}/api/v1/chat/ws?token=${encodeURIComponent(token ?? "")}`
  );
};
