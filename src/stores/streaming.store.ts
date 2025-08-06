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
  
  // Actions
  startStreaming: (connectionId: string, messageId: string) => void;
  appendToStream: (content: string) => void;
  finishStreaming: () => void;
  clearStreaming: () => void;
  setStreamingMessage: (message: StreamingMessage | null) => void;
}

export const useStreamingStore = create<StreamingState>((set, get) => ({
  // State
  streamingMessage: null,
  isStreaming: false,
  streamingConnectionId: null,
  streamBuffer: '',
  
  // Actions
  startStreaming: (connectionId, messageId) => set({
    isStreaming: true,
    streamingConnectionId: connectionId,
    streamBuffer: '',
    streamingMessage: {
      id: messageId,
      content: '',
      role: 'assistant',
      isStreaming: true,
      connectionId,
    },
  }),
  
  appendToStream: (content) => set((state) => {
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
  }),
  
  setStreamingMessage: (message) => set({ streamingMessage: message }),
}));