import { create } from 'zustand';

/**
 * Streaming Store - Handles real-time streaming state
 * This is client-side state that doesn't come from REST APIs
 */

interface StreamingMessage {
  id: string;
  content: string;
  role: 'assistant';
  isStreaming: boolean;
  connectionId?: string;
  model?: string;
}

interface StreamingState {
  // Current streaming message
  streamingMessage: StreamingMessage | null;
  
  // Streaming status
  isStreaming: boolean;
  streamingConnectionId: string | null;
  
  // Buffer for accumulating streamed content
  streamBuffer: string;
  
  // Abort controller reference
  abortController: AbortController | null;
  
  // Error state
  streamError: string | null;
  
  // Context usage tracking
  contextUsage: {
    totalMessages: number;
    includedMessages: number;
    characters: number;
    maxCharacters: number;
    truncated: boolean;
  } | null;
  
  // Actions
  startStreaming: (connectionId: string, messageId: string) => void;
  appendToStream: (content: string) => void;
  finishStreaming: () => void;
  clearStreaming: () => void;
  setStreamingMessage: (message: StreamingMessage | null) => void;
  setAbortController: (controller: AbortController | null) => void;
  abortStream: () => void;
  setStreamError: (error: string | null) => void;
  setContextUsage: (usage: StreamingState['contextUsage']) => void;
}

export const useStreamingStore = create<StreamingState>((set, get) => ({
  // State
  streamingMessage: null,
  isStreaming: false,
  streamingConnectionId: null,
  streamBuffer: '',
  abortController: null,
  streamError: null,
  contextUsage: null,
  
  // Actions
  startStreaming: (connectionId, messageId) => set({
    isStreaming: true,
    streamingConnectionId: connectionId,
    streamBuffer: '',
    streamError: null, // Clear any previous errors
    streamingMessage: {
      id: messageId,
      content: '',
      role: 'assistant',
      isStreaming: true,
      connectionId,
    },
  }),
  
  appendToStream: (content) => set((state) => {
    // Preserve formatting by ensuring we don't lose whitespace/newlines
    // when concatenating chunks
    const newBuffer = state.streamBuffer + content;
    return {
      streamBuffer: newBuffer,
      streamingMessage: state.streamingMessage ? {
        ...state.streamingMessage,
        content: newBuffer,
      } : null,
    };
  }),
  
  finishStreaming: () => set((state) => ({
    isStreaming: false,
    streamingConnectionId: null,
    streamBuffer: '',
    streamingMessage: state.streamingMessage ? {
      ...state.streamingMessage,
      isStreaming: false,
    } : null,
  })),
  
  clearStreaming: () => set({
    streamingMessage: null,
    isStreaming: false,
    streamingConnectionId: null,
    streamBuffer: '',
    abortController: null,
    streamError: null,
  }),
  
  setStreamingMessage: (message) => set({ streamingMessage: message }),
  
  setAbortController: (controller) => set({ abortController: controller }),
  
  setStreamError: (error) => set({ streamError: error }),
  
  setContextUsage: (usage) => set({ contextUsage: usage }),
  
  abortStream: () => {
    const state = get();
    if (state.abortController) {
      console.log('[StreamingStore] Aborting stream');
      state.abortController.abort();
    }
    // Clear the streaming state
    set({
      streamingMessage: null,
      isStreaming: false,
      streamingConnectionId: null,
      streamBuffer: '',
      abortController: null,
      streamError: null,
    });
  },
}));