import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/api-client';

// Types
export interface ApiKey {
  id: string;
  user_id: string;
  provider: 'openai' | 'anthropic' | 'google' | 'groq' | 'openrouter' | 'deepseek';
  key_hash: string;
  key_hint: string; // Last 4 characters of the key
  label?: string;
  is_active: boolean;
  last_used_at?: string;
  created_at: string;
  updated_at: string;
  metadata?: Record<string, unknown>;
}

export interface CreateApiKeyRequest {
  provider: ApiKey['provider'];
  api_key: string;
  label?: string;
  metadata?: Record<string, unknown>;
}

export interface UpdateApiKeyRequest {
  id: string;
  label?: string;
  is_active?: boolean;
  metadata?: Record<string, unknown>;
}

// Query keys
export const apiKeyKeys = {
  all: ['apiKeys'] as const,
  lists: () => [...apiKeyKeys.all, 'list'] as const,
  list: (provider?: string) => [...apiKeyKeys.lists(), { provider }] as const,
  detail: (id: string) => [...apiKeyKeys.all, 'detail', id] as const,
};

/**
 * Hook to get all API keys for the current user
 */
export const useApiKeys = (provider?: string) => {
  return useQuery({
    queryKey: apiKeyKeys.list(provider),
    queryFn: async () => {
      const params = provider ? `?provider=${provider}` : '';
      return apiClient.get<ApiKey[]>(`/api-keys${params}`);
    },
    staleTime: 30 * 1000, // Consider fresh for 30 seconds
  });
};

/**
 * Hook to get a single API key
 */
export const useApiKey = (id: string) => {
  return useQuery({
    queryKey: apiKeyKeys.detail(id),
    queryFn: async () => {
      return apiClient.get<ApiKey>(`/api-keys/${id}`);
    },
    enabled: !!id,
  });
};

/**
 * Hook to create a new API key
 */
export const useCreateApiKey = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: CreateApiKeyRequest) => {
      return apiClient.post<ApiKey>('/api-keys', data);
    },
    onSuccess: (newKey) => {
      // Invalidate and refetch API keys list
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.lists() });
      
      // Optionally update the cache directly
      queryClient.setQueryData<ApiKey[]>(
        apiKeyKeys.list(),
        (old) => [...(old || []), newKey]
      );
    },
  });
};

/**
 * Hook to update an API key
 */
export const useUpdateApiKey = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, ...data }: UpdateApiKeyRequest) => {
      return apiClient.put<ApiKey>(`/api-keys/${id}`, data);
    },
    onSuccess: (updatedKey) => {
      // Update the specific key in cache
      queryClient.setQueryData(apiKeyKeys.detail(updatedKey.id), updatedKey);
      
      // Update the list cache
      queryClient.setQueryData<ApiKey[]>(
        apiKeyKeys.list(),
        (old) => old?.map(key => key.id === updatedKey.id ? updatedKey : key) || []
      );
      
      // Invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.lists() });
    },
  });
};

/**
 * Hook to delete an API key
 */
export const useDeleteApiKey = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      return apiClient.delete(`/api-keys/${id}`);
    },
    onSuccess: (_, deletedId) => {
      // Remove from cache
      queryClient.setQueryData<ApiKey[]>(
        apiKeyKeys.list(),
        (old) => old?.filter(key => key.id !== deletedId) || []
      );
      
      // Invalidate queries
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.lists() });
      queryClient.removeQueries({ queryKey: apiKeyKeys.detail(deletedId) });
    },
  });
};

/**
 * Hook to validate an API key
 */
export const useValidateApiKey = () => {
  return useMutation({
    mutationFn: async ({ provider, api_key }: { provider: string; api_key: string }) => {
      return apiClient.post<{ valid: boolean; error?: string }>(
        '/api-keys/validate',
        { provider, api_key }
      );
    },
  });
};

/**
 * Hook to test an API key with a provider
 */
export const useTestApiKey = () => {
  return useMutation({
    mutationFn: async (id: string) => {
      return apiClient.post<{ 
        success: boolean; 
        response?: unknown; 
        error?: string;
        latency_ms?: number;
      }>(`/api-keys/${id}/test`);
    },
  });
};