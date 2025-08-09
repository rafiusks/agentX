import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/api-client';
import { useStreamingStore } from '../../stores/streaming.store';

// Types
export interface Chat {
  ID: string;
  UserID?: string;
  Title: string;
  Provider?: string | null;
  Model?: string | null;
  CreatedAt?: string;
  UpdatedAt?: string;
  Metadata?: any;
  // Legacy fields for compatibility
  id?: string;
  title?: string;
}

export interface Message {
  id: string;
  chat_id: string;
  role: 'user' | 'assistant' | 'system' | 'function';
  content: string;
  provider?: string;
  model?: string;
  created_at: string;
  metadata?: Record<string, any>;
  // Optional streaming/UI properties
  isStreaming?: boolean;
  functionCall?: {
    name: string;
    arguments: string;
  };
}

export interface CreateChatRequest {
  title?: string;
  provider: string;
  model: string;
  initial_message?: string;
  metadata?: Record<string, any>;
}

export interface SendMessageRequest {
  chat_id: string;
  content: string;
  connection_id?: string;
  stream?: boolean;
}

export interface StreamResponse {
  id: string;
  delta: string;
  finish_reason?: 'stop' | 'length' | 'error';
}

// Query keys
export const chatKeys = {
  all: ['sessions'] as const,
  lists: () => [...chatKeys.all, 'list'] as const,
  list: (filters?: { provider?: string; search?: string }) => 
    [...chatKeys.lists(), { filters }] as const,
  detail: (id: string) => [...chatKeys.all, 'detail', id] as const,
  messages: (chatId: string) => [...chatKeys.all, 'messages', chatId] as const,
};

/**
 * Hook to get all chats for the current user
 */
export const useChats = (filters?: { provider?: string; search?: string }) => {
  return useQuery({
    queryKey: chatKeys.list(filters),
    queryFn: async () => {
      const params = new URLSearchParams();
      if (filters?.provider) params.append('provider', filters.provider);
      if (filters?.search) params.append('search', filters.search);
      
      const queryString = params.toString();
      const response = await apiClient.get<{ sessions: Chat[] }>(`/sessions${queryString ? `?${queryString}` : ''}`);
      
      // Sort sessions by updated/created date (newest first) at the query level
      const sessions = response.sessions || [];
      
      return sessions.sort((a, b) => {
        // Use UpdatedAt if available, otherwise fall back to CreatedAt
        const dateA = new Date(a.UpdatedAt || a.CreatedAt || 0).getTime();
        const dateB = new Date(b.UpdatedAt || b.CreatedAt || 0).getTime();
        return dateB - dateA; // Newest first
      });
    },
    staleTime: 30 * 1000, // Consider fresh for 30 seconds to reduce re-fetching
  });
};

/**
 * Hook to get a single chat
 */
export const useChat = (id?: string) => {
  return useQuery({
    queryKey: chatKeys.detail(id!),
    queryFn: async () => {
      return apiClient.get<Chat>(`/sessions/${id}`);
    },
    enabled: !!id,
  });
};

/**
 * Hook to get messages for a chat
 */
export const useChatMessages = (chatId?: string) => {
  return useQuery({
    queryKey: chatKeys.messages(chatId!),
    queryFn: async () => {
      const response = await apiClient.get<{ messages: Message[] }>(`/sessions/${chatId}/messages`);
      return response.messages || [];
    },
    enabled: !!chatId,
    staleTime: 5 * 1000, // Consider fresh for 5 seconds
  });
};

/**
 * Hook to create a new chat
 */
export const useCreateChat = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: CreateChatRequest) => {
      return apiClient.post<Chat>('/sessions', data);
    },
    onSuccess: (newChat) => {
      // Optimistically add the new chat to the list instead of invalidating
      queryClient.setQueryData<Chat[]>(chatKeys.list(), (old) => {
        if (!old) return [newChat];
        // Add new chat at the beginning (newest first)
        return [newChat, ...old];
      });
    },
  });
};

/**
 * Hook to update a chat
 */
export const useUpdateChat = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, ...data }: { id: string; title?: string; metadata?: any }) => {
      return apiClient.put<Chat>(`/sessions/${id}`, data);
    },
    onSuccess: (updatedChat) => {
      // Update specific chat in cache
      queryClient.setQueryData(chatKeys.detail(updatedChat.id || updatedChat.ID), updatedChat);
      
      // Invalidate all chat lists to refetch with updated data
      queryClient.invalidateQueries({ queryKey: chatKeys.lists() });
    },
  });
};

/**
 * Hook to delete a chat
 */
export const useDeleteChat = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      return apiClient.delete(`/sessions/${id}`);
    },
    onSuccess: (_, deletedId) => {
      // Invalidate all chat queries to refetch
      queryClient.invalidateQueries({ queryKey: chatKeys.all });
      
      // Remove specific queries for deleted chat
      queryClient.removeQueries({ queryKey: chatKeys.detail(deletedId) });
      queryClient.removeQueries({ queryKey: chatKeys.messages(deletedId) });
    },
  });
};

/**
 * Hook to send a message (non-streaming)
 */
export const useSendMessage = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ chat_id, content, connection_id }: SendMessageRequest) => {
      // First, add the user message to cache
      const userMessage: Message = {
        id: `temp-${Date.now()}`,
        chat_id,
        role: 'user',
        content,
        created_at: new Date().toISOString(),
      };
      
      queryClient.setQueryData<Message[]>(
        chatKeys.messages(chat_id),
        (old) => [...(old || []), userMessage]
      );
      
      // Send to server
      return apiClient.post<Message>(`/sessions/${chat_id}/messages`, {
        content,
        connection_id,
      });
    },
    onSuccess: (assistantMessage, variables) => {
      // Add assistant message to cache
      queryClient.setQueryData<Message[]>(
        chatKeys.messages(variables.chat_id),
        (old) => [...(old || []), assistantMessage]
      );
      
      // Invalidate chat detail and list to update timestamps
      queryClient.invalidateQueries({ queryKey: chatKeys.detail(variables.chat_id) });
      queryClient.invalidateQueries({ queryKey: chatKeys.lists() });
    },
  });
};

/**
 * Hook to send a streaming message
 */
export const useSendStreamingMessage = () => {
  const queryClient = useQueryClient();
  const { startStreaming, appendToStream, finishStreaming } = useStreamingStore();

  return useMutation({
    mutationFn: async ({ chat_id, content, connection_id }: SendMessageRequest) => {
      console.log('[useSendStreamingMessage] Starting with:', { chat_id, content, connection_id });
      
      // Add user message to cache
      const userMessage: Message = {
        id: `temp-${Date.now()}`,
        chat_id,
        role: 'user',
        content,
        created_at: new Date().toISOString(),
      };
      
      queryClient.setQueryData<Message[]>(
        chatKeys.messages(chat_id),
        (old) => [...(old || []), userMessage]
      );
      
      // Start streaming
      const messageId = `stream-${Date.now()}`;
      startStreaming(connection_id || 'default', messageId);
      
      // Get existing messages for context
      const existingMessages = queryClient.getQueryData<Message[]>(chatKeys.messages(chat_id)) || [];
      const messages = [
        ...existingMessages
          .filter(m => {
            // Exclude the temporary user message we just added
            if (m.id === userMessage.id) return false;
            // Only include messages with valid roles
            return m.role && ['user', 'assistant', 'system'].includes(m.role) && m.content;
          })
          .map(m => ({
            role: m.role,
            content: m.content
          })),
        { role: 'user', content }
      ];
      
      // Use the streaming endpoint
      console.log('[useSendStreamingMessage] Sending streaming request with:', { 
        session_id: chat_id, 
        messages_count: messages.length,
        messages,
        connection_id 
      });
      
      return apiClient.stream(`/chat/stream`, {
        session_id: chat_id,
        messages,
        preferences: {
          connection_id
        }
      }, (chunk: any) => {
        // Handle OpenAI-formatted streaming chunks
        console.log('Stream chunk:', chunk);
        
        // Skip invalid chunks
        if (!chunk || typeof chunk !== 'object') {
          console.warn('Invalid chunk received:', chunk);
          return;
        }
        
        if (chunk.choices && chunk.choices[0]) {
          const choice = chunk.choices[0];
          
          // Handle content delta
          if (choice.delta && choice.delta.content) {
            appendToStream(choice.delta.content);
          }
          
          // Handle finish - only process if we have accumulated content
          else if (choice.finish_reason === 'stop') {
            const content = useStreamingStore.getState().streamBuffer;
            if (content && content.trim()) {
              // Add complete message to cache
              const assistantMessage: Message = {
                id: chunk.id || `msg-${Date.now()}`,
                chat_id,
                role: 'assistant',
                content,
                created_at: new Date().toISOString(),
              };
              
              queryClient.setQueryData<Message[]>(
                chatKeys.messages(chat_id),
                (old) => [...(old || []), assistantMessage]
              );
              
              // Invalidate chat list to update timestamps
              queryClient.invalidateQueries({ queryKey: chatKeys.lists() });
            }
            
            // Clear streaming state after adding to cache
            useStreamingStore.getState().clearStreaming();
          }
        }
      }, (error: Error) => {
        console.error('Stream error:', error);
        finishStreaming();
      });
    },
    onError: (error) => {
      console.error('Mutation error:', error);
      finishStreaming();
    },
  });
};

/**
 * Hook to clear chat history
 */
export const useClearChatHistory = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (chatId: string) => {
      return apiClient.delete(`/chats/${chatId}/messages`);
    },
    onSuccess: (_, chatId) => {
      // Clear messages from cache
      queryClient.setQueryData(chatKeys.messages(chatId), []);
      
      // Invalidate chat to update message_count
      queryClient.invalidateQueries({ queryKey: chatKeys.detail(chatId) });
    },
  });
};