import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/api-client';
import { useStreamingStore } from '../../stores/streaming.store';
import { formatLLMError } from '../../utils/error-formatter';
import { getContextConfig } from '../../config/context';
import { selectImportantMessages } from '../../utils/message-importance';
import { useContextStore } from '../../stores/context.store';
import { buildContextWithSummaries } from './useSummaries';
import { usePreferencesStore } from '../../stores/preferences.store';
import { withRetry, withRateLimitRetry } from '../../utils/retry';

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
  // Importance scoring
  importance?: number; // 0-1 score for message importance
  importanceFlags?: {
    hasCode?: boolean;
    hasError?: boolean;
    hasDecision?: boolean;
    isUserCorrection?: boolean;
    tokens?: number;
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
    refetchOnWindowFocus: false, // Don't refetch on window focus
    refetchOnMount: false, // Don't refetch if data exists
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
    staleTime: 60 * 1000, // Consider fresh for 60 seconds
    refetchOnWindowFocus: false, // Don't refetch on window focus
    refetchOnMount: false, // Don't refetch if data exists
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
  const { startStreaming, appendToStream, finishStreaming, setAbortController, setStreamError, setContextUsage } = useStreamingStore();
  const { config: userContextConfig } = useContextStore();
  const { getSystemPrompt, maxResponseTokens } = usePreferencesStore();
  
  // Track timeout for stuck streams and abort controller
  let streamTimeoutId: ReturnType<typeof setTimeout> | null = null;
  let abortController: AbortController | null = null;

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
      
      // Create abort controller for this stream
      abortController = new AbortController();
      setAbortController(abortController); // Store it in the streaming store
      
      // Start streaming
      const messageId = `stream-${Date.now()}`;
      startStreaming(connection_id || 'default', messageId);
      
      // Set a timeout to force-close stuck streams (3 minutes should be enough for most responses)
      streamTimeoutId = setTimeout(() => {
        console.warn('[useSendStreamingMessage] Stream timeout - forcing stream close');
        const currentState = useStreamingStore.getState();
        if (currentState.isStreaming) {
          // Abort the request to stop backend processing
          if (abortController) {
            abortController.abort();
            abortController = null;
          }
          
          // Save any accumulated content before clearing
          const content = currentState.streamBuffer;
          if (content && content.trim()) {
            const assistantMessage: Message = {
              id: `msg-timeout-${Date.now()}`,
              chat_id,
              role: 'assistant',
              content: content,
              created_at: new Date().toISOString(),
            };
            
            queryClient.setQueryData<Message[]>(
              chatKeys.messages(chat_id),
              (old) => [...(old || []), assistantMessage]
            );
          }
          
          // Force clear the streaming state
          useStreamingStore.getState().clearStreaming();
        }
      }, 180000); // 3 minute timeout
      
      // Get existing messages for context with sliding window
      const existingMessages = queryClient.getQueryData<Message[]>(chatKeys.messages(chat_id)) || [];
      
      // Fetch summaries for this session (synchronously from cache)
      const summariesData = queryClient.getQueryData<any[]>(['summaries', 'session', chat_id]) || [];
      
      // Get provider type from current connection (if available)
      // This is a simplified approach - you might want to get this from the connection data
      const providerType = 'openai-compatible'; // Default, should be fetched from connection
      
      // Merge user preferences with provider defaults
      const providerConfig = getContextConfig(providerType);
      const contextConfig = {
        ...providerConfig,
        ...userContextConfig, // User preferences override provider defaults
      };
      
      const MAX_CONTEXT_MESSAGES = contextConfig.maxMessages;
      const MAX_CONTEXT_CHARS = contextConfig.maxCharacters;
      
      // Filter valid messages
      const validMessages = existingMessages
        .filter(m => {
          // Exclude the temporary user message we just added
          if (m.id === userMessage.id) return false;
          // Only include messages with valid roles and content
          // Exclude error messages from context
          if (m.content?.startsWith('❌ Error:')) return false;
          return m.role && ['user', 'assistant', 'system'].includes(m.role) && m.content;
        });
      
      // Choose context selection strategy
      let contextMessages: Message[];
      let totalChars: number;
      let includedSummary = false;
      
      // Try to use summaries if we have them and many messages
      if (summariesData.length > 0 && validMessages.length > MAX_CONTEXT_MESSAGES) {
        const summaryResult = buildContextWithSummaries(
          validMessages,
          summariesData,
          MAX_CONTEXT_MESSAGES,
          MAX_CONTEXT_CHARS
        );
        
        contextMessages = summaryResult.contextMessages;
        includedSummary = summaryResult.includedSummary;
        totalChars = contextMessages.reduce((sum, m) => sum + (m.content?.length || 0), 0);
      } 
      // Use importance scoring if enabled (smart mode) and no summary
      else if (contextConfig.useImportanceScoring && contextConfig.strategy === 'smart') {
        // Use importance-based selection
        contextMessages = selectImportantMessages(
          validMessages,
          MAX_CONTEXT_MESSAGES,
          MAX_CONTEXT_CHARS / 4 // Convert chars to approximate tokens
        );
        totalChars = contextMessages.reduce((sum, m) => sum + (m.content?.length || 0), 0);
      } else {
        // Use sliding window (default)
        contextMessages = validMessages.slice(-MAX_CONTEXT_MESSAGES);
        
        // Further trim if total content is too long
        totalChars = contextMessages.reduce((sum, m) => sum + (m.content?.length || 0), 0);
        while (totalChars > MAX_CONTEXT_CHARS && contextMessages.length > 2) {
          // Remove oldest messages until we're under the limit
          contextMessages = contextMessages.slice(1);
          totalChars = contextMessages.reduce((sum, m) => sum + (m.content?.length || 0), 0);
        }
      }
      
      // Track context usage
      const contextUsage = {
        totalMessages: validMessages.length,
        includedMessages: contextMessages.length,
        characters: totalChars,
        maxCharacters: MAX_CONTEXT_CHARS,
        truncated: validMessages.length > contextMessages.length,
        usingSummary: includedSummary
      };
      
      // Set context usage in store for UI display
      setContextUsage(contextUsage);
      
      // Log context usage for debugging
      console.log(`[Context] Using ${contextMessages.length}/${validMessages.length} messages (${totalChars} chars)${includedSummary ? ' with summary' : ''}`);
      
      // Build the final message array
      const messages = [];
      
      // Add system prompt based on user preferences
      messages.push({
        role: 'system',
        content: getSystemPrompt()
      });
      
      // If we have a summary, it's already included in contextMessages
      // Otherwise add a truncation note if needed
      if (!includedSummary && contextUsage.truncated) {
        messages.push({
          role: 'system',
          content: `[Note: Previous ${validMessages.length - contextMessages.length} messages omitted to fit context window]`
        });
      }
      
      // Add the context messages (which may include summary)
      messages.push(
        ...contextMessages.map(m => ({
          role: m.role,
          content: m.content
        })),
        { role: 'user', content }
      );
      
      // Use the streaming endpoint
      console.log('[useSendStreamingMessage] Sending streaming request with:', { 
        session_id: chat_id, 
        messages_count: messages.length,
        messages,
        connection_id 
      });
      
      // Build request with optional max tokens
      const requestBody: any = {
        session_id: chat_id,
        messages,
        preferences: {
          connection_id
        }
      };
      
      // Add max tokens if set
      if (maxResponseTokens) {
        requestBody.max_tokens = maxResponseTokens;
      }
      
      // Wrap streaming call with retry logic for rate limits
      return withRateLimitRetry(
        async () => {
          return await apiClient.stream(`/chat/stream`, requestBody, (chunk: any) => {
        // Handle OpenAI-formatted streaming chunks
        // Skip invalid chunks
        if (!chunk || typeof chunk !== 'object') {
          console.warn('Invalid chunk received:', chunk);
          return;
        }
        
        // Check for error in chunk
        if (chunk.error) {
          console.error('Stream chunk error:', chunk.error);
          const errorMessage = formatLLMError(chunk.error);
          setStreamError(errorMessage);
          
          // Add error to chat
          const errorMsg: Message = {
            id: `error-chunk-${Date.now()}`,
            chat_id,
            role: 'assistant', 
            content: `❌ Error: ${errorMessage}`,
            created_at: new Date().toISOString(),
          };
          
          queryClient.setQueryData<Message[]>(
            chatKeys.messages(chat_id),
            (old) => [...(old || []), errorMsg]
          );
          
          // Clear streaming
          useStreamingStore.getState().clearStreaming();
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
            console.log('[useSendStreamingMessage] Received finish_reason: stop');
            
            // Clear the timeout since stream completed normally
            if (streamTimeoutId) {
              clearTimeout(streamTimeoutId);
              streamTimeoutId = null;
            }
            
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
            console.log('[useSendStreamingMessage] Clearing streaming state');
            useStreamingStore.getState().clearStreaming();
          }
        }
      }, (error: Error) => {
        console.error('Stream error:', error);
        
        // Set error in the store so it can be displayed
        const errorMessage = formatLLMError(error);
        setStreamError(errorMessage);
        
        // Add error message to chat
        const errorMsg: Message = {
          id: `error-${Date.now()}`,
          chat_id,
          role: 'assistant',
          content: `❌ Error: ${errorMessage}`,
          created_at: new Date().toISOString(),
        };
        
        queryClient.setQueryData<Message[]>(
          chatKeys.messages(chat_id),
          (old) => [...(old || []), errorMsg]
        );
        
        // Clear timeout on error
        if (streamTimeoutId) {
          clearTimeout(streamTimeoutId);
          streamTimeoutId = null;
        }
        finishStreaming();
      }, () => {
        // onComplete callback - called when stream ends (either by [DONE] or naturally)
        console.log('[useSendStreamingMessage] Stream completed via onComplete callback');
        
        // Clear timeout since stream completed
        if (streamTimeoutId) {
          clearTimeout(streamTimeoutId);
          streamTimeoutId = null;
        }
        
        // Save any accumulated content
        const currentState = useStreamingStore.getState();
        const content = currentState.streamBuffer;
        
        if (content && content.trim() && currentState.isStreaming) {
          // Add complete message to cache if not already added
          const assistantMessage: Message = {
            id: `msg-complete-${Date.now()}`,
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
        
        // Clear streaming state
        useStreamingStore.getState().clearStreaming();
      }, abortController?.signal); // Pass the abort signal to enable cancellation
        },
        (attempt, delay) => {
          // Notify user about retry
          const retryMsg: Message = {
            id: `retry-${Date.now()}`,
            chat_id,
            role: 'system',
            content: `⏳ Rate limited. Retrying in ${Math.round(delay / 1000)}s... (Attempt ${attempt})`,
            created_at: new Date().toISOString(),
          };
          
          queryClient.setQueryData<Message[]>(
            chatKeys.messages(chat_id),
            (old) => [...(old || []), retryMsg]
          );
        }
      ); // End of withRateLimitRetry
    },
    onError: (error, variables) => {
      console.error('Mutation error:', error);
      
      // Extract and format error message
      const errorMessage = formatLLMError(error);
      
      // Set error in store
      setStreamError(errorMessage);
      
      // Add error message to chat
      const errorMsg: Message = {
        id: `error-${Date.now()}`,
        chat_id: variables.chat_id,
        role: 'assistant',
        content: `❌ Error: ${errorMessage}`,
        created_at: new Date().toISOString(),
      };
      
      queryClient.setQueryData<Message[]>(
        chatKeys.messages(variables.chat_id),
        (old) => [...(old || []), errorMsg]
      );
      
      // Clear timeout on error
      if (streamTimeoutId) {
        clearTimeout(streamTimeoutId);
        streamTimeoutId = null;
      }
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