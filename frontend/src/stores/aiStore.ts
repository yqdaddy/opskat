import { create } from "zustand";
import {
  SendAIMessage,
  SetAIProvider,
  DetectLocalCLIs,
  GetInitContext,
  ResetAISession,
} from "../../wailsjs/go/main/App";
import { ai } from "../../wailsjs/go/models";
import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
import { useAssetStore } from "./assetStore";
import i18n from "../i18n";

export interface ChatMessage {
  role: "user" | "assistant" | "tool";
  content: string;
  toolCalls?: ai.ToolCall[];
  streaming?: boolean;
}

interface AIState {
  messages: ChatMessage[];
  conversationId: string;
  configured: boolean;
  sending: boolean;
  localCLIs: ai.CLIInfo[];

  configure: (
    providerType: string,
    apiBase: string,
    apiKey: string,
    model: string
  ) => Promise<void>;
  send: (content: string) => Promise<void>;
  detectCLIs: () => Promise<void>;
  clear: () => void;
}

let eventCleanup: (() => void) | null = null;

export const useAIStore = create<AIState>((set, get) => ({
  messages: [],
  conversationId: "conv-1",
  configured: false,
  sending: false,
  localCLIs: [],

  configure: async (providerType, apiBase, apiKey, model) => {
    await SetAIProvider(providerType, apiBase, apiKey, model);
    set({ configured: true });
  },

  send: async (content) => {
    const state = get();
    if (state.sending) return;

    let actualContent = content;

    // 拦截 /init 命令
    if (content.trim() === "/init") {
      const { selectedAssetId, selectedGroupId } = useAssetStore.getState();
      if (!selectedAssetId && !selectedGroupId) {
        // 添加提示消息
        set({
          messages: [
            ...state.messages,
            { role: "user", content: "/init" },
            {
              role: "assistant",
              content: i18n.t("ai.initNoSelection"),
              streaming: false,
            },
          ],
        });
        return;
      }
      try {
        actualContent = await GetInitContext(
          selectedAssetId || 0,
          selectedGroupId || 0
        );
      } catch (e) {
        set({
          messages: [
            ...state.messages,
            { role: "user", content: "/init" },
            {
              role: "assistant",
              content: `${i18n.t("ai.initError")}: ${e}`,
              streaming: false,
            },
          ],
        });
        return;
      }
    }

    // 添加用户消息（UI 显示原始输入）
    const displayContent = content.trim() === "/init" ? "/init" : content;
    const userMsg: ChatMessage = { role: "user", content: displayContent };
    const newMessages = [...state.messages, userMsg];
    set({ messages: newMessages, sending: true });

    // 添加空的 assistant 消息（用于流式填充）
    const assistantMsg: ChatMessage = {
      role: "assistant",
      content: "",
      streaming: true,
    };
    set({ messages: [...newMessages, assistantMsg] });

    // 监听流式事件
    const convId = state.conversationId;
    const eventName = "ai:event:" + convId;

    if (eventCleanup) {
      eventCleanup();
    }

    EventsOn(eventName, (event: { type: string; content?: string; error?: string }) => {
      const msgs = get().messages;
      const lastIdx = msgs.length - 1;

      switch (event.type) {
        case "content":
          if (lastIdx >= 0 && msgs[lastIdx].role === "assistant") {
            const updated = [...msgs];
            updated[lastIdx] = {
              ...updated[lastIdx],
              content: updated[lastIdx].content + (event.content || ""),
            };
            set({ messages: updated });
          }
          break;
        case "done":
          if (lastIdx >= 0) {
            const updated = [...msgs];
            updated[lastIdx] = { ...updated[lastIdx], streaming: false };
            set({ messages: updated, sending: false });
          }
          EventsOff(eventName);
          eventCleanup = null;
          break;
        case "error":
          if (lastIdx >= 0) {
            const updated = [...msgs];
            updated[lastIdx] = {
              ...updated[lastIdx],
              content:
                updated[lastIdx].content + `\n\n**Error:** ${event.error}`,
              streaming: false,
            };
            set({ messages: updated, sending: false });
          }
          EventsOff(eventName);
          eventCleanup = null;
          break;
      }
    });

    eventCleanup = () => EventsOff(eventName);

    // 转换为后端消息格式（/init 时发送实际上下文而非 "/init"）
    const apiMessages = newMessages.map((m, idx) => {
      const msgContent = idx === newMessages.length - 1 ? actualContent : m.content;
      return new ai.Message({
        role: m.role,
        content: msgContent,
      });
    });

    try {
      await SendAIMessage(convId, apiMessages);
    } catch (e) {
      set({ sending: false });
      EventsOff(eventName);
    }
  },

  detectCLIs: async () => {
    const clis = await DetectLocalCLIs();
    set({ localCLIs: clis || [] });
  },

  clear: () => {
    set({ messages: [], sending: false });
    if (eventCleanup) {
      eventCleanup();
      eventCleanup = null;
    }
    // 重置后端会话
    ResetAISession().catch(() => {});
  },
}));

// 应用启动时自动恢复 AI 配置
const providerType = localStorage.getItem("ai_provider_type");
if (providerType) {
  const apiBase = localStorage.getItem("ai_api_base") || "";
  const apiKey = localStorage.getItem("ai_api_key") || "";
  const model = localStorage.getItem("ai_model") || "";
  useAIStore.getState().configure(providerType, apiBase, apiKey, model).catch(() => {});
}
