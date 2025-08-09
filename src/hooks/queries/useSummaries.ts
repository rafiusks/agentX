import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/api-client';

export interface SessionSummary {
  id: string;
  session_id: string;
  summary_text: string;
  message_count: number;
  start_message_id?: string;
  end_message_id?: string;
  tokens_saved: number;
  model_used: string;
  created_at: string;
}

// Query keys
const summaryKeys = {
  all: ['summaries'] as const,
  bySession: (sessionId: string) => [...summaryKeys.all, 'session', sessionId] as const,
};

/**
 * Hook to fetch summaries for a session
 */
export const useSessionSummaries = (sessionId?: string) => {
  return useQuery({
    queryKey: summaryKeys.bySession(sessionId || ''),
    queryFn: async () => {
      if (!sessionId) return [];
      const response = await apiClient.get<SessionSummary[]>(`/sessions/${sessionId}/summaries`);
      return response;
    },
    enabled: !!sessionId,
  });
};

/**
 * Hook to generate a summary for a session
 */
export const useGenerateSummary = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ 
      sessionId, 
      messageCount = 20 
    }: { 
      sessionId: string; 
      messageCount?: number;
    }) => {
      const response = await apiClient.post<SessionSummary>(
        `/sessions/${sessionId}/summary`,
        { message_count: messageCount }
      );
      return response;
    },
    onSuccess: (data) => {
      // Invalidate summaries cache
      queryClient.invalidateQueries({ 
        queryKey: summaryKeys.bySession(data.session_id) 
      });
    },
  });
};

/**
 * Build context with summaries
 */
export function buildContextWithSummaries(
  messages: any[],
  summaries: SessionSummary[],
  maxMessages: number,
  maxChars: number
): { 
  contextMessages: any[];
  includedSummary: boolean;
  summaryText?: string;
} {
  // If we have many messages, use summary for older ones
  if (messages.length > maxMessages && summaries.length > 0) {
    // Get the most recent summary
    const latestSummary = summaries[0];
    
    // Find messages after the summary
    const summaryEndTime = new Date(latestSummary.created_at);
    const recentMessages = messages.filter(m => 
      new Date(m.created_at) > summaryEndTime
    );
    
    // If we have enough recent messages, use summary + recent
    if (recentMessages.length >= 5) {
      return {
        contextMessages: [
          {
            role: 'system',
            content: `[Previous conversation summary: ${latestSummary.summary_text}]`
          },
          ...recentMessages.slice(-(maxMessages - 1)) // -1 for the summary message
        ],
        includedSummary: true,
        summaryText: latestSummary.summary_text
      };
    }
  }
  
  // Otherwise, just use recent messages
  return {
    contextMessages: messages.slice(-maxMessages),
    includedSummary: false
  };
}