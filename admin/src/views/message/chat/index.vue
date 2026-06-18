<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref
} from "vue";
import { IconifyIconOnline } from "@/components/ReIcon";
import { message } from "@/utils/message";
import { getMe } from "@/api/user";
import { useI18n } from "@/i18n";
import {
  createChatSocket,
  getChatHistory,
  getChatUsers,
  markChatRead,
  translateChatText,
  uploadChatFile,
  type ChatMessage,
  type ChatMessageType,
  type ChatUser
} from "@/api/chat";

type ChatSocketPayload = {
  type?: string;
  user_id?: number;
  online?: boolean;
  error?: string;
  message?: ChatMessage;
  read_msg_ids?: number[];
};

type SpeechRecognitionConstructor = new () => SpeechRecognition;
type SpeechRecognitionEvent = Event & {
  results: ArrayLike<{
    0?: {
      transcript?: string;
    };
  }>;
};
type SpeechRecognitionErrorEvent = Event & {
  error: string;
};
type SpeechRecognition = EventTarget & {
  lang: string;
  continuous: boolean;
  interimResults: boolean;
  start: () => void;
  stop: () => void;
  onresult: ((event: SpeechRecognitionEvent) => void) | null;
  onerror: ((event: SpeechRecognitionErrorEvent) => void) | null;
  onend: (() => void) | null;
};

declare global {
  interface Window {
    SpeechRecognition?: SpeechRecognitionConstructor;
    webkitSpeechRecognition?: SpeechRecognitionConstructor;
  }
}

const { t, locale } = useI18n();
const ui = new Proxy({} as Record<string, string>, {
  get: (_, key: string) => t(`chat.${key}`)
});

const emojiOptions = [
  "😀",
  "😁",
  "😂",
  "😊",
  "😍",
  "😎",
  "🤝",
  "👍",
  "👏",
  "🎉",
  "🔥",
  "💡",
  "✅",
  "❤️",
  "🙏",
  "🚀"
];

const loading = ref(false);
const uploading = ref(false);
const users = ref<ChatUser[]>([]);
const activeUser = ref<ChatUser>();
const chatVisible = ref(false);
const messages = ref<ChatMessage[]>([]);
const input = ref("");
const currentUserId = ref(0);
const currentUsername = ref("");
const socket = ref<WebSocket>();
const messageBodyRef = ref<HTMLElement>();
const socketReady = ref(false);
const reconnecting = ref(false);
const emojiVisible = ref(false);
const imageInputRef = ref<HTMLInputElement>();
const videoInputRef = ref<HTMLInputElement>();
const fileInputRef = ref<HTMLInputElement>();
const mediaElements = new Map<string, HTMLMediaElement>();
const playingMediaKey = ref("");
const recording = ref(false);
const recordSeconds = ref(0);
const liveTranscript = ref("");
const transcripts = reactive<Record<number, string>>({});
const translations = reactive<Record<number, string>>({});
const translating = reactive<Record<number, boolean>>({});
// 未读消息数（key=对方userId）
const unreadCounts = reactive<Record<number, number>>({});
// 已被对方读取的消息id集合（对方打开聊天窗口时我们收到通知）
const readByPeer = reactive<Record<number, boolean>>({});

const heartbeatIntervalMs = 15000;
const heartbeatTimeoutMs = 10000;
const maxReconnectDelayMs = 30000;
let heartbeatTimer: number | undefined;
let heartbeatTimeout: number | undefined;
let reconnectTimer: number | undefined;
let reconnectAttempt = 0;
let allowReconnect = true;
let mediaRecorder: MediaRecorder | undefined;
let audioChunks: Blob[] = [];
let recordTimer: number | undefined;
let recognition: SpeechRecognition | undefined;
let chatPageMounted = false;

const activeTitle = computed(() => activeUser.value?.username ?? ui.chat);
const activeSubtitle = computed(() =>
  activeUser.value?.online ? ui.online : ui.offline
);
const socketStatusText = computed(() => {
  if (socketReady.value) return ui.connected;
  return reconnecting.value ? ui.reconnecting : ui.disconnected;
});
const chatUsers = computed(() =>
  users.value.filter(user => user.id !== currentUserId.value)
);
const searchKeyword = ref("");
const filteredUsers = computed(() => {
  const kw = searchKeyword.value.trim().toLowerCase();
  if (!kw) return chatUsers.value;
  return chatUsers.value.filter(
    u =>
      u.username.toLowerCase().includes(kw) ||
      (u.email ?? "").toLowerCase().includes(kw) ||
      (u.phone ?? "").toLowerCase().includes(kw)
  );
});

const apiBase = import.meta.env.VITE_API_BASE_URL || "";

const scrollToBottom = (immediate = false) => {
  const doScroll = () => {
    const el = messageBodyRef.value;
    if (el) el.scrollTop = el.scrollHeight;
  };
  nextTick(() => {
    doScroll();
    if (immediate) return;
    // 媒体（图片/视频）加载完成后再次滚动
    const el = messageBodyRef.value;
    if (!el) return;
    const media = el.querySelectorAll("img, video");
    media.forEach(m => {
      const handler = () => doScroll();
      m.addEventListener("load", handler, { once: true });
      m.addEventListener("loadedmetadata", handler, { once: true });
      m.addEventListener("error", handler, { once: true });
    });
    // 兜底：100ms / 300ms / 600ms 再滚一次
    setTimeout(doScroll, 100);
    setTimeout(doScroll, 300);
    setTimeout(doScroll, 600);
  });
};

const resolveMediaUrl = (url: string) => {
  if (!url) return "";
  if (/^https?:\/\//i.test(url)) return url;
  if (!apiBase) return url;
  return new URL(url, apiBase).href;
};

const formatTime = (value: string) => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
};

const formatFileSize = (size = 0) => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
};

const formatDuration = (seconds: number) => {
  const minute = Math.floor(seconds / 60)
    .toString()
    .padStart(2, "0");
  const second = Math.floor(seconds % 60)
    .toString()
    .padStart(2, "0");
  return `${minute}:${second}`;
};

const messageTypeOf = (item: ChatMessage): ChatMessageType => {
  return item.message_type || "text";
};

const senderName = (item: ChatMessage) => {
  if (item.from_user_id === currentUserId.value) return ui.me;
  return activeUser.value?.username ?? ui.chat;
};

const avatarText = (item: ChatMessage) => {
  const name =
    item.from_user_id === currentUserId.value
      ? currentUsername.value || ui.me
      : activeUser.value?.username || ui.chat;
  return name.slice(0, 1).toUpperCase();
};

const mediaKey = (item: ChatMessage) => {
  return `${item.id || item.created_at}-${item.message_type}`;
};

const getTranscript = (item: ChatMessage) => {
  return transcripts[item.id] || item.transcript || item.content || "";
};

const getTranslation = (item: ChatMessage) => {
  return translations[item.id] || item.translation || "";
};

const registerMedia = (key: string, el: HTMLMediaElement | null) => {
  if (!el) {
    mediaElements.delete(key);
    return;
  }
  mediaElements.set(key, el);
  el.onplay = () => {
    playingMediaKey.value = key;
    for (const [otherKey, media] of mediaElements) {
      if (otherKey !== key) media.pause();
    }
  };
  el.onpause = () => {
    if (playingMediaKey.value === key) playingMediaKey.value = "";
  };
  el.onended = () => {
    if (playingMediaKey.value === key) playingMediaKey.value = "";
  };
};

const registerTemplateMedia = (key: string, el: Element | null) => {
  registerMedia(key, el instanceof HTMLMediaElement ? el : null);
};

const toggleMedia = (key: string) => {
  const media = mediaElements.get(key);
  if (!media) return;
  if (media.paused) {
    media.play();
  } else {
    media.pause();
  }
};

const loadUsers = async () => {
  loading.value = true;
  try {
    const res = await getChatUsers();
    users.value = res.users ?? [];
    // 用后端返回的未读数初始化（仅对未打开的对话）
    users.value.forEach(u => {
      if (u.id !== activeUser.value?.id && (u.unread_count ?? 0) > 0) {
        unreadCounts[u.id] = u.unread_count ?? 0;
      }
    });
    if (activeUser.value) {
      activeUser.value =
        users.value.find(user => user.id === activeUser.value?.id) ??
        activeUser.value;
    }
  } catch (error) {
    message(ui.userLoadFailed, { type: "error" });
  } finally {
    loading.value = false;
  }
};

const refreshMe = async () => {
  const res = await getMe();
  currentUserId.value = res.data.id;
  currentUsername.value = res.data.username;
};

const clearHeartbeat = () => {
  if (heartbeatTimer) {
    window.clearInterval(heartbeatTimer);
    heartbeatTimer = undefined;
  }
  if (heartbeatTimeout) {
    window.clearTimeout(heartbeatTimeout);
    heartbeatTimeout = undefined;
  }
};

const clearReconnect = () => {
  if (reconnectTimer) {
    window.clearTimeout(reconnectTimer);
    reconnectTimer = undefined;
  }
};

function scheduleReconnect() {
  if (!allowReconnect || reconnectTimer) return;
  reconnecting.value = true;
  const delay = Math.min(
    1000 * 2 ** Math.min(reconnectAttempt, 5),
    maxReconnectDelayMs
  );
  reconnectAttempt += 1;
  reconnectTimer = window.setTimeout(() => {
    reconnectTimer = undefined;
    connectSocket();
  }, delay);
}

const startHeartbeat = (ws: WebSocket) => {
  clearHeartbeat();
  const sendHeartbeat = () => {
    if (socket.value !== ws || ws.readyState !== WebSocket.OPEN) return;
    try {
      ws.send(JSON.stringify({ type: "ping" }));
    } catch (error) {
      ws.close();
      return;
    }
    if (heartbeatTimeout) window.clearTimeout(heartbeatTimeout);
    heartbeatTimeout = window.setTimeout(() => {
      if (socket.value === ws && ws.readyState === WebSocket.OPEN) ws.close();
    }, heartbeatTimeoutMs);
  };
  sendHeartbeat();
  heartbeatTimer = window.setInterval(sendHeartbeat, heartbeatIntervalMs);
};

const closeCurrentSocket = () => {
  const ws = socket.value;
  if (!ws) return;
  ws.onopen = null;
  ws.onclose = null;
  ws.onerror = null;
  ws.onmessage = null;
  if (
    ws.readyState === WebSocket.OPEN ||
    ws.readyState === WebSocket.CONNECTING
  ) {
    ws.close();
  }
};

function connectSocket() {
  clearReconnect();
  clearHeartbeat();
  closeCurrentSocket();
  const ws = createChatSocket();
  if (!ws) {
    socket.value = undefined;
    socketReady.value = false;
    reconnecting.value = false;
    return;
  }
  socket.value = ws;
  socketReady.value = false;

  ws.onopen = () => {
    if (socket.value !== ws) return;
    socketReady.value = true;
    reconnecting.value = false;
    reconnectAttempt = 0;
    startHeartbeat(ws);
    loadUsers();
  };
  ws.onclose = () => {
    if (socket.value !== ws) return;
    socketReady.value = false;
    clearHeartbeat();
    scheduleReconnect();
  };
  ws.onerror = () => {
    if (socket.value !== ws) return;
    socketReady.value = false;
    ws.close();
  };
  ws.onmessage = event => {
    let payload: ChatSocketPayload;
    try {
      payload = JSON.parse(event.data);
    } catch (error) {
      return;
    }
    if (payload.type === "pong") {
      if (heartbeatTimeout) {
        window.clearTimeout(heartbeatTimeout);
        heartbeatTimeout = undefined;
      }
      return;
    }
    if (payload.type === "online") {
      users.value = users.value.map(user =>
        user.id === payload.user_id
          ? { ...user, online: Boolean(payload.online) }
          : user
      );
      if (activeUser.value?.id === payload.user_id) {
        activeUser.value = {
          ...activeUser.value,
          online: Boolean(payload.online)
        };
      }
      return;
    }
    if (payload.type === "read") {
      // 对方已读了我发的消息
      if (payload.read_msg_ids?.length) {
        payload.read_msg_ids.forEach(id => { readByPeer[id] = true; });
      }
      return;
    }
    if (payload.type === "error") {
      message(payload.error || ui.messageFailed, { type: "error" });
      return;
    }
    if (payload.type === "message") {
      if (!payload.message) return;
      const msg = payload.message;
      const activeId = activeUser.value?.id;
      const belongsToOpenChat =
        activeId &&
        ((msg.from_user_id === currentUserId.value &&
          msg.to_user_id === activeId) ||
          (msg.from_user_id === activeId &&
            msg.to_user_id === currentUserId.value));

      if (belongsToOpenChat) {
        messages.value.push(msg);
        scrollToBottom();
        // 对方发来消息且当前聊天窗口打开，立即标记为已读（通知后端，后端会推送 read 给对方）
        if (msg.from_user_id !== currentUserId.value && activeId) {
          markChatRead(activeId).catch(() => {});
        }
      } else if (msg.from_user_id !== currentUserId.value) {
        // 后台收到消息，累加未读数
        const senderId = msg.from_user_id;
        unreadCounts[senderId] = (unreadCounts[senderId] || 0) + 1;
      }
      if (msg.from_user_id !== currentUserId.value) loadUsers();
    }
  };
}

const ensureSocketReady = () => {
  if (socket.value?.readyState === WebSocket.OPEN) return true;
  message(ui.socketNotReady, { type: "warning" });
  scheduleReconnect();
  return false;
};

const openChat = async (user: ChatUser) => {
  activeUser.value = user;
  chatVisible.value = true;
  emojiVisible.value = false;
  // 清零未读数
  unreadCounts[user.id] = 0;
  try {
    const res = await getChatHistory(user.id);
    messages.value = res.messages ?? [];
    // 从历史消息中初始化已读状态
    messages.value.forEach(m => {
      if (m.from_user_id === currentUserId.value && m.is_read) {
        readByPeer[m.id] = true;
      }
    });
    // 标记对方发给我的消息为已读（通知后端）
    markChatRead(user.id).catch(() => {});
    // 清零本地未读数
    unreadCounts[user.id] = 0;
    for (const item of messages.value) {
      if (item.transcript) transcripts[item.id] = item.transcript;
      if (item.translation) translations[item.id] = item.translation;
    }
    scrollToBottom();
  } catch (error) {
    message(ui.historyLoadFailed, { type: "error" });
  }
};

const sendSocketMessage = (payload: Record<string, unknown>) => {
  if (!activeUser.value || !ensureSocketReady()) return false;
  try {
    socket.value?.send(
      JSON.stringify({
        type: "message",
        to_user_id: activeUser.value.id,
        ...payload
      })
    );
    return true;
  } catch (error) {
    message(ui.messageSendFailed, { type: "error" });
    return false;
  }
};

const sendTextMessage = () => {
  const content = input.value.trim();
  if (!content) return;
  if (
    sendSocketMessage({
      message_type: "text",
      content
    })
  ) {
    input.value = "";
    emojiVisible.value = false;
  }
};

const sendEmoji = (emoji: string) => {
  if (
    sendSocketMessage({
      message_type: "emoji",
      content: emoji
    })
  ) {
    emojiVisible.value = false;
  }
};

const triggerPicker = (type: "image" | "video" | "file") => {
  if (!activeUser.value) return;
  const pickerMap = {
    image: imageInputRef,
    video: videoInputRef,
    file: fileInputRef
  };
  pickerMap[type].value?.click();
};

const sendMediaMessage = async (
  file: File,
  preferredType?: "image" | "video" | "audio" | "file",
  extra: Record<string, unknown> = {}
) => {
  if (!activeUser.value || !ensureSocketReady()) return false;
  uploading.value = true;
  try {
    const caption = input.value.trim();
    const result = await uploadChatFile(file);
    const messageType = preferredType ?? result.message_type;
    const sent = sendSocketMessage({
      message_type: messageType,
      content: caption,
      media_url: result.url,
      file_name: result.file_name,
      mime_type: result.mime_type,
      file_size: result.file_size,
      ...extra
    });
    if (sent) {
      input.value = "";
      emojiVisible.value = false;
    }
    return sent;
  } catch (error) {
    message(ui.fileSendFailed, { type: "error" });
    return false;
  } finally {
    uploading.value = false;
  }
};

const handlePickedFile = async (
  event: Event,
  preferredType?: "image" | "video" | "file"
) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  target.value = "";
  if (!file) return;
  await sendMediaMessage(file, preferredType);
};

const createSpeechRecognition = () => {
  const Recognition =
    window.SpeechRecognition || window.webkitSpeechRecognition;
  if (!Recognition) return undefined;
  const instance = new Recognition();
  instance.lang = locale.value;
  instance.continuous = true;
  instance.interimResults = true;
  return instance;
};

const startRecordTimer = () => {
  recordSeconds.value = 0;
  if (recordTimer) window.clearInterval(recordTimer);
  recordTimer = window.setInterval(() => {
    recordSeconds.value += 1;
  }, 1000);
};

const stopRecordTimer = () => {
  if (recordTimer) {
    window.clearInterval(recordTimer);
    recordTimer = undefined;
  }
};

const stopSpeechRecognition = () => {
  if (!recognition) return;
  try {
    recognition.onend = null;
    recognition.stop();
  } catch (error) {
    // 部分浏览器在识别已经结束时再次 stop 会抛错，不应影响录音上传。
  } finally {
    recognition = undefined;
  }
};

const startVoiceInput = async () => {
  if (!activeUser.value) return;
  if (!navigator.mediaDevices?.getUserMedia) {
    message(ui.unsupportedRecorder, { type: "warning" });
    return;
  }
  if (typeof MediaRecorder === "undefined") {
    message(ui.unsupportedRecorderCodec, { type: "warning" });
    return;
  }
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    const mimeType = MediaRecorder.isTypeSupported("audio/webm")
      ? "audio/webm"
      : "";
    mediaRecorder = new MediaRecorder(stream, mimeType ? { mimeType } : {});
    audioChunks = [];
    liveTranscript.value = "";
    recording.value = true;
    startRecordTimer();

    recognition = createSpeechRecognition();
    if (recognition) {
      recognition.onresult = event => {
        let text = "";
        for (let i = 0; i < event.results.length; i += 1) {
          text += event.results[i][0]?.transcript || "";
        }
        liveTranscript.value = text.trim();
      };
      recognition.onerror = () => {
        message(ui.speechUnavailable, { type: "warning" });
      };
      recognition.start();
    }

    mediaRecorder.ondataavailable = event => {
      if (event.data.size > 0) audioChunks.push(event.data);
    };
    mediaRecorder.onstop = async () => {
      stream.getTracks().forEach(track => track.stop());
      stopRecordTimer();
      stopSpeechRecognition();
      recording.value = false;
      const blob = new Blob(audioChunks, {
        type: mediaRecorder?.mimeType || "audio/webm"
      });
      audioChunks = [];
      if (blob.size <= 0) {
        message(ui.emptyVoice, { type: "warning" });
        return;
      }
      const file = new File([blob], `voice-${Date.now()}.webm`, {
        type: blob.type
      });
      const sent = await sendMediaMessage(file, "audio", {
        content: liveTranscript.value || ui.voiceReady,
        transcript: liveTranscript.value
      });
      message(sent ? ui.voiceSent : ui.voiceSendFailed, {
        type: sent ? "success" : "error"
      });
      liveTranscript.value = "";
      recordSeconds.value = 0;
    };
    mediaRecorder.start();
  } catch (error) {
    recording.value = false;
    stopRecordTimer();
    stopSpeechRecognition();
    message(ui.micDenied, { type: "error" });
  }
};

const stopVoiceInput = () => {
  if (!mediaRecorder || mediaRecorder.state === "inactive") return;
  try {
    if (mediaRecorder.state === "recording") {
      mediaRecorder.requestData();
    }
  } catch (error) {
    // requestData 只是尽量提前刷出数据，失败时仍交给 stop 触发 dataavailable。
  }
  mediaRecorder.stop();
};

const toggleVoiceInput = () => {
  if (recording.value) {
    stopVoiceInput();
  } else {
    startVoiceInput();
  }
};

const transcribeMessage = (item: ChatMessage) => {
  const text = getTranscript(item);
  if (!text) {
    message(ui.noTranscript, { type: "warning" });
    return;
  }
  transcripts[item.id] = text;
};

const translateMessage = async (item: ChatMessage) => {
  const text = getTranscript(item);
  if (!text) {
    message(ui.noTranslatableText, { type: "warning" });
    return;
  }
  translating[item.id] = true;
  try {
    const res = await translateChatText(text, "zh");
    translations[item.id] = res.translated_text;
  } catch (error) {
    message(ui.translateFailed, { type: "error" });
  } finally {
    translating[item.id] = false;
  }
};

const closeChat = () => {
  chatVisible.value = false;
  messages.value = [];
  activeUser.value = undefined;
  emojiVisible.value = false;
  for (const media of mediaElements.values()) media.pause();
  mediaElements.clear();
  playingMediaKey.value = "";
};

onMounted(async () => {
  chatPageMounted = true;
  await refreshMe();
  if (!chatPageMounted) return;
  await loadUsers();
  if (!chatPageMounted) return;
  connectSocket();
});

onBeforeUnmount(() => {
  chatPageMounted = false;
  allowReconnect = false;
  if (recording.value) stopVoiceInput();
  stopRecordTimer();
  stopSpeechRecognition();
  clearReconnect();
  clearHeartbeat();
  closeCurrentSocket();
});
</script>


<template>
  <div class="im-layout">
    <!-- 左侧面板 -->
    <div class="im-sidebar">
      <div class="im-sidebar__topbar">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索"
          clearable
          size="small"
          class="im-search-input"
        >
          <template #prefix>
            <IconifyIconOnline icon="ri:search-line" style="font-size:14px;color:#999" />
          </template>
        </el-input>
        <el-tooltip :content="ui.refresh" placement="bottom">
          <button type="button" class="im-icon-btn" :disabled="loading" @click="loadUsers">
            <IconifyIconOnline icon="ri:refresh-line" />
          </button>
        </el-tooltip>
      </div>

      <div class="im-conn-bar" :class="socketReady ? 'connected' : 'disconnected'">
        <span class="im-conn-dot" />
        {{ socketStatusText }}
      </div>

      <div v-loading="loading" class="im-user-list">
        <template v-if="filteredUsers.length">
          <button
            v-for="user in filteredUsers"
            :key="user.id"
            type="button"
            class="im-user-item"
            :class="{ active: activeUser?.id === user.id }"
            @click="openChat(user)"
          >
            <div class="im-avatar" :class="{ 'is-online': user.online }">
              {{ user.username.slice(0, 1).toUpperCase() }}
            </div>
            <div class="im-user-info">
              <div class="im-user-name">{{ user.username }}</div>
              <div class="im-user-sub">{{ user.email || user.phone }}</div>
            </div>
            <div class="im-user-meta">
              <span class="im-online-badge" :class="{ online: user.online }">
                {{ user.online ? ui.online : ui.offline }}
              </span>
              <span v-if="unreadCounts[user.id] > 0" class="im-unread-badge">
                {{ unreadCounts[user.id] > 99 ? '99+' : unreadCounts[user.id] }}
              </span>
            </div>
          </button>
        </template>
        <el-empty v-else-if="!loading" :description="ui.empty" :image-size="72" />
      </div>
    </div>

    <!-- 右侧主区域 -->
    <div class="im-main">
      <div v-if="!activeUser" class="im-empty-state">
        <IconifyIconOnline icon="ri:chat-smile-2-line" class="im-empty-icon" />
        <p>{{ ui.chat }}</p>
        <small>{{ ui.pageDesc }}</small>
      </div>

      <template v-else>
        <div class="im-chat-header">
          <div class="im-avatar sm">{{ activeTitle.slice(0, 1).toUpperCase() }}</div>
          <div class="im-chat-header__info">
            <strong>{{ activeTitle }}</strong>
            <span :class="activeUser?.online ? 'is-online' : ''">
              <i class="dot" />{{ activeSubtitle }}
            </span>
          </div>
        </div>

        <div ref="messageBodyRef" class="im-messages">
          <template v-if="messages.length">
            <div
              v-for="item in messages"
              :key="item.id || `${item.from_user_id}-${item.created_at}`"
              class="im-msg-row"
              :class="{ mine: item.from_user_id === currentUserId }"
            >
              <div class="im-msg-avatar">{{ avatarText(item) }}</div>
              <div class="im-msg-body">
                <div class="im-msg-meta">
                  <span>{{ senderName(item) }}</span>
                  <time>{{ formatTime(item.created_at) }}</time>
                </div>
                <div class="im-bubble" :class="`type-${messageTypeOf(item)}`">
                  <template v-if="messageTypeOf(item) === 'image'">
                    <el-image
                      class="im-media-img"
                      fit="cover"
                      preview-teleported
                      :src="resolveMediaUrl(item.media_url)"
                      :preview-src-list="[resolveMediaUrl(item.media_url)]"
                    />
                    <p v-if="item.content" class="im-caption">{{ item.content }}</p>
                  </template>

                  <template v-else-if="messageTypeOf(item) === 'video'">
                    <div class="im-video-wrap">
                      <video
                        :ref="el => registerTemplateMedia(mediaKey(item), el as Element | null)"
                        class="im-media-video"
                        :src="resolveMediaUrl(item.media_url)"
                        preload="metadata"
                        playsinline
                      />
                      <button type="button" class="im-play-btn" @click="toggleMedia(mediaKey(item))">
                        <IconifyIconOnline :icon="playingMediaKey === mediaKey(item) ? 'ri:pause-fill' : 'ri:play-fill'" />
                      </button>
                    </div>
                    <p v-if="item.content" class="im-caption">{{ item.content }}</p>
                  </template>

                  <template v-else-if="messageTypeOf(item) === 'audio'">
                    <div class="im-voice">
                      <button type="button" class="im-voice-btn" @click="toggleMedia(mediaKey(item))">
                        <IconifyIconOnline :icon="playingMediaKey === mediaKey(item) ? 'ri:pause-fill' : 'ri:mic-fill'" />
                      </button>
                      <div class="im-voice-wave" />
                      <span class="im-voice-label">{{ item.content || ui.voiceReady }}</span>
                      <audio
                        :ref="el => registerTemplateMedia(mediaKey(item), el as Element | null)"
                        :src="resolveMediaUrl(item.media_url)"
                        preload="metadata"
                      />
                    </div>
                    <div class="im-msg-tools">
                      <button type="button" @click="transcribeMessage(item)">{{ ui.transcribe }}</button>
                      <button type="button" :disabled="translating[item.id]" @click="translateMessage(item)">{{ ui.translate }}</button>
                    </div>
                    <p v-if="getTranscript(item)" class="im-note"><strong>{{ ui.transcript }}：</strong>{{ getTranscript(item) }}</p>
                    <p v-if="getTranslation(item)" class="im-note"><strong>{{ ui.translation }}：</strong>{{ getTranslation(item) }}</p>
                  </template>

                  <template v-else-if="messageTypeOf(item) === 'file'">
                    <a class="im-file" :href="resolveMediaUrl(item.media_url)" target="_blank" rel="noopener">
                      <IconifyIconOnline icon="ri:file-3-line" class="im-file-icon" />
                      <span>
                        <strong>{{ item.file_name || ui.file }}</strong>
                        <small>{{ formatFileSize(item.file_size) }}</small>
                      </span>
                    </a>
                    <p v-if="item.content" class="im-caption">{{ item.content }}</p>
                  </template>

                  <template v-else>{{ item.content }}</template>
                </div>
                <div
                  v-if="item.from_user_id === currentUserId && item.id"
                  class="im-read-status"
                  :class="{ read: readByPeer[item.id] || item.is_read }"
                >
                  {{ (readByPeer[item.id] || item.is_read) ? ui.read : ui.unread }}
                </div>
              </div>
            </div>
          </template>
          <el-empty v-else :description="ui.empty" :image-size="80" style="margin-top:60px" />
        </div>

        <div v-if="recording" class="im-recording-bar">
          <span class="im-rec-dot" />
          {{ ui.recording }} {{ formatDuration(recordSeconds) }}
          <small v-if="liveTranscript">{{ liveTranscript }}</small>
        </div>

        <div v-if="emojiVisible" class="im-emoji-panel">
          <button
            v-for="emoji in emojiOptions"
            :key="emoji"
            type="button"
            class="im-emoji-btn"
            @click="sendEmoji(emoji)"
          >{{ emoji }}</button>
        </div>

        <div class="im-composer">
          <div class="im-composer__tools">
            <el-tooltip :content="ui.emoji" placement="top">
              <button type="button" class="im-tool-btn" :class="{ active: emojiVisible }" @click="emojiVisible = !emojiVisible">
                <IconifyIconOnline icon="ri:emotion-line" />
              </button>
            </el-tooltip>
            <el-tooltip :content="recording ? ui.stopRecord : ui.record" placement="top">
              <button type="button" class="im-tool-btn" :class="{ recording }" @click="toggleVoiceInput">
                <IconifyIconOnline :icon="recording ? 'ri:stop-fill' : 'ri:mic-line'" />
              </button>
            </el-tooltip>
            <el-tooltip :content="ui.image" placement="top">
              <button type="button" class="im-tool-btn" :disabled="uploading" @click="triggerPicker('image')">
                <IconifyIconOnline icon="ri:image-line" />
              </button>
            </el-tooltip>
            <el-tooltip :content="ui.video" placement="top">
              <button type="button" class="im-tool-btn" :disabled="uploading" @click="triggerPicker('video')">
                <IconifyIconOnline icon="ri:video-line" />
              </button>
            </el-tooltip>
            <el-tooltip :content="ui.file" placement="top">
              <button type="button" class="im-tool-btn" :disabled="uploading" @click="triggerPicker('file')">
                <IconifyIconOnline icon="ri:attachment-2" />
              </button>
            </el-tooltip>
          </div>
          <el-input
            v-model="input"
            type="textarea"
            :rows="4"
            resize="none"
            maxlength="500"
            :placeholder="ui.placeholder"
            class="im-textarea"
            @keydown.enter.exact.prevent="sendTextMessage"
          />
          <div class="im-composer__footer">
            <span class="im-char-count">{{ input.length }}/500</span>
            <el-button
              type="primary"
              size="default"
              :disabled="!input.trim() && !uploading"
              :loading="uploading"
              @click="sendTextMessage"
            >{{ ui.send }}</el-button>
          </div>
        </div>
      </template>
    </div><!-- /im-main -->

    <input ref="imageInputRef" hidden type="file" accept="image/*" @change="handlePickedFile($event, 'image')" />
    <input ref="videoInputRef" hidden type="file" accept="video/*" @change="handlePickedFile($event, 'video')" />
    <input ref="fileInputRef" hidden type="file" @change="handlePickedFile($event, 'file')" />
  </div><!-- /im-layout -->
</template>

<style scoped>
.im-layout {
  width: 100%;
  height: calc(100vh - 138px);
  min-height: 560px;
  display: flex;
  overflow: hidden;
  background: #f4f6f9;
  z-index: 1;
}

.im-sidebar {
  display: flex;
  flex-direction: column;
  width: 280px;
  flex-shrink: 0;
  background: #fff;
  border-right: 1px solid #edf0f5;
}

.im-sidebar__topbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 12px 10px;
  border-bottom: 1px solid #edf0f5;
}

.im-search-input { flex: 1; }

.im-search-input :deep(.el-input__wrapper) {
  background: #f4f6f9;
  border-radius: 20px !important;
  box-shadow: none !important;
  border: 1px solid transparent;
}

.im-search-input :deep(.el-input__wrapper:hover),
.im-search-input :deep(.el-input__wrapper.is-focus) {
  border-color: #c0c4cc;
}

.im-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 6px;
  background: transparent;
  border: none;
  cursor: pointer;
  color: #909399;
  font-size: 16px;
  transition: background 0.15s;
}

.im-icon-btn:hover { background: #f0f4ff; color: #2563eb; }

.im-conn-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 14px;
  font-size: 11.5px;
  font-weight: 500;
}

.im-conn-bar.connected { color: #22c55e; background: #f0fdf4; }
.im-conn-bar.disconnected { color: #f59e0b; background: #fffbeb; }

.im-conn-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.im-user-list {
  flex: 1;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: #e5e7eb transparent;
}

.im-user-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 14px;
  background: transparent;
  border: none;
  border-bottom: 1px solid #f5f5f5;
  cursor: pointer;
  text-align: left;
  transition: background 0.14s;
}

.im-user-item:hover { background: #f5f7ff; }
.im-user-item.active { background: #eff3ff; }

.im-avatar {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border-radius: 8px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff;
  font-size: 15px;
  font-weight: 700;
}

.im-avatar.sm {
  width: 34px;
  height: 34px;
  font-size: 13px;
  border-radius: 6px;
}

.im-avatar.is-online::after {
  content: '';
  position: absolute;
  bottom: -2px;
  right: -2px;
  width: 10px;
  height: 10px;
  background: #22c55e;
  border-radius: 50%;
  border: 2px solid #fff;
}

.im-user-info { flex: 1; min-width: 0; }

.im-user-name {
  font-size: 13.5px;
  font-weight: 600;
  color: rgb(0 0 0 / 82%);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.im-user-sub {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.im-user-meta {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}
.im-online-badge { font-size: 11px; color: #9ca3af; }
.im-online-badge.online { color: #22c55e; }

/* 未读角标 */
.im-unread-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  font-size: 11px;
  font-weight: 700;
  color: #fff;
  background: #ef4444;
  border-radius: 9px;
  line-height: 1;
}

/* 已读/未读状态 */
.im-read-status {
  display: flex;
  color: #666;
  font-size: 12px;
  justify-content: flex-end;
  margin-top: 3px;
  padding-right: 2px;
}

.im-read-text {
  font-size: 11px;
}

.im-read-text.read {
  color: #22c55e;
}

.im-read-text.unread {
  color: #9ca3af;
}

.im-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
  background: #f7f8fa;
  overflow: hidden;
}

.im-empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #9ca3af;
  gap: 10px;
}

.im-empty-icon { font-size: 52px; color: #d1d5db; }
.im-empty-state p { font-size: 16px; font-weight: 600; color: #6b7280; margin: 0; }
.im-empty-state small { font-size: 13px; }

.im-chat-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 20px;
  background: #fff;
  border-bottom: 1px solid #edf0f5;
  flex-shrink: 0;
}

.im-chat-header__info strong {
  display: block;
  font-size: 14.5px;
  font-weight: 700;
  color: rgb(0 0 0 / 85%);
}

.im-chat-header__info span {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: #9ca3af;
  margin-top: 2px;
}

.im-chat-header__info span.is-online { color: #22c55e; }

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  display: inline-block;
  flex-shrink: 0;
}

.im-messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 20px 24px;
  scrollbar-width: thin;
  scrollbar-color: #d1d5db transparent;
}

.im-messages::-webkit-scrollbar { width: 5px; }
.im-messages::-webkit-scrollbar-thumb { background: #d1d5db; border-radius: 3px; }
.im-messages::-webkit-scrollbar-track { background: transparent; }

.im-msg-row {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  margin-bottom: 18px;
}

.im-msg-row.mine { flex-direction: row-reverse; }

.im-msg-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  flex-shrink: 0;
  border-radius: 8px;
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff;
  font-size: 13px;
  font-weight: 700;
}

.im-msg-row.mine .im-msg-avatar {
  background: linear-gradient(135deg, #10b981, #059669);
}

.im-msg-body { max-width: min(68%, 560px); }

.im-msg-meta {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 4px;
  font-size: 11.5px;
  color: #9ca3af;
}

.im-msg-row.mine .im-msg-meta { justify-content: flex-end; }

.im-bubble {
  width: fit-content;
  max-width: 100%;
  padding: 10px 14px;
  color: #1f2937;
  white-space: pre-wrap;
  word-break: break-word;
  background: #fff;
  border-radius: 4px 12px 12px 12px;
  box-shadow: 0 1px 4px rgb(0 0 0 / 7%);
  font-size: 14px;
  line-height: 1.55;
}

.im-msg-row.mine .im-bubble {
  color: #fff;
  background: #2563eb;
  border-radius: 12px 4px 12px 12px;
  box-shadow: 0 2px 8px rgb(37 99 235 / 28%);
}

.im-bubble.type-emoji {
  font-size: 32px;
  padding: 2px 6px;
  background: transparent;
  box-shadow: none;
  line-height: 1.2;
}

.im-msg-row.mine .im-bubble.type-emoji { background: transparent; box-shadow: none; }
.im-caption { margin: 6px 0 0; font-size: 12.5px; opacity: 0.8; }

.im-media-img {
  display: block;
  max-width: min(280px, 50vw);
  max-height: 240px;
  border-radius: 8px;
  object-fit: cover;
}

.im-video-wrap {
  position: relative;
  border-radius: 10px;
  overflow: hidden;
  background: #0f172a;
}

.im-media-video {
  display: block;
  width: min(280px, 50vw);
  max-height: 200px;
  object-fit: contain;
}

.im-play-btn {
  position: absolute;
  bottom: 10px;
  right: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: rgb(0 0 0 / 55%);
  color: #fff;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  font-size: 18px;
}

.im-voice {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 200px;
}

.im-voice-btn {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: #22c55e;
  color: #fff;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  font-size: 16px;
  flex-shrink: 0;
}

.im-msg-row.mine .im-voice-btn { background: rgba(255, 255, 255, 0.3); }

.im-voice-wave {
  flex: 1;
  height: 20px;
  background: repeating-linear-gradient(90deg, currentColor 0 3px, transparent 3px 7px);
  opacity: 0.4;
  border-radius: 2px;
}

.im-voice-label {
  font-size: 12px;
  opacity: 0.75;
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.im-msg-tools { display: flex; gap: 8px; margin-top: 6px; }

.im-msg-tools button {
  font-size: 12px;
  color: #6b7280;
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 3px;
  text-decoration: underline;
}

.im-msg-tools button:hover { color: #2563eb; }

.im-msg-row.mine .im-msg-tools button {
  color: rgba(255, 255, 255, 0.75);
  text-decoration-color: rgba(255, 255, 255, 0.4);
}

.im-msg-row.mine .im-msg-tools button:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.15);
}

.im-msg-row.mine .im-note { color: rgba(255, 255, 255, 0.85); }
.im-msg-row.mine .im-note strong { color: #fff; }
.im-note { font-size: 12px; margin: 4px 0 0; opacity: 0.82; }

.im-file {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 200px;
  color: inherit;
  text-decoration: none;
  padding: 4px 0;
}

.im-file-icon { font-size: 28px; flex-shrink: 0; opacity: 0.75; }

.im-file strong,
.im-file small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.im-file small { margin-top: 2px; font-size: 11.5px; opacity: 0.65; }

.im-recording-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 16px;
  background: #fff0f0;
  color: #ef4444;
  font-size: 13px;
  font-weight: 500;
  flex-shrink: 0;
  border-top: 1px solid #fecaca;
}

.im-rec-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ef4444;
  animation: pulse 1s ease-in-out infinite;
}

.im-recording-bar small {
  color: #6b7280;
  font-weight: 400;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.im-emoji-panel {
  display: grid;
  grid-template-columns: repeat(10, 34px);
  gap: 4px;
  padding: 10px 14px;
  background: #fff;
  border-top: 1px solid #edf0f5;
  flex-shrink: 0;
}

.im-emoji-btn {
  width: 34px;
  height: 34px;
  font-size: 20px;
  line-height: 1;
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.12s;
}

.im-emoji-btn:hover { background: #f0f4ff; }

.im-composer {
  background: #fff;
  border-top: 1px solid #edf0f5;
  flex-shrink: 0;
}

.im-composer__tools {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 8px 14px 4px;
  border-bottom: 1px solid #f5f5f5;
}

.im-tool-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 18px;
  color: #6b7280;
  transition: background 0.14s, color 0.14s;
}

.im-tool-btn:hover,
.im-tool-btn.active { background: #f0f4ff; color: #2563eb; }
.im-tool-btn.recording { background: #fff0f0; color: #ef4444; }
.im-tool-btn:disabled { opacity: 0.4; cursor: not-allowed; }

.im-textarea :deep(.el-textarea__inner) {
  border: none !important;
  box-shadow: none !important;
  border-radius: 0 !important;
  padding: 10px 16px;
  font-size: 14px;
  line-height: 1.55;
  resize: none;
  background: transparent;
}

.im-composer__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  padding: 6px 14px 10px;
}

.im-char-count { font-size: 12px; color: #c0c4cc; }

@keyframes pulse {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.4); opacity: 0.5; }
}

@media (max-width: 700px) {
  .im-layout { flex-direction: column; height: auto; }
  .im-sidebar { width: 100%; max-height: 40vh; }
  .im-emoji-panel { grid-template-columns: repeat(6, 34px); }
}
</style>
