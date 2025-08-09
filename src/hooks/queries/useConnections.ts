import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/api-client';

// Types
export interface Connection {
  id: string;
  user_id: string;
  provider: 'openai' | 'anthropic' | 'google' | 'groq' | 'openrouter' | 'deepseek' | 'ollama' | 'local';
  name: string;
  api_key_id?: string;
  base_url?: string;
  is_active: boolean;
  is_default: boolean;
  settings?: {
    temperature?: number;
    max_tokens?: number;
    top_p?: number;
    frequency_penalty?: number;
    presence_penalty?: number;
    timeout?: number;
    [key: string]: any;
  };
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
  last_used_at?: string;
}

export interface Model {
  id: string;
  name: string;
  display_name?: string;
  provider: string;
  context_window?: number;
  max_output_tokens?: number;
  supports_functions?: boolean;
  supports_vision?: boolean;
  supports_streaming?: boolean;
  pricing?: {
    input_cost_per_1k?: number;
    output_cost_per_1k?: number;
  };
}

export interface CreateConnectionRequest {
  provider: Connection['provider'];
  name: string;
  api_key_id?: string;
  base_url?: string;
  is_default?: boolean;
  settings?: Connection['settings'];
  metadata?: Record<string, any>;
}

export interface UpdateConnectionRequest {
  id: string;
  name?: string;
  api_key_id?: string;
  base_url?: string;
  is_active?: boolean;
  is_default?: boolean;
  settings?: Connection['settings'];
  metadata?: Record<string, any>;
}

// Query keys
export const connectionKeys = {
  all: ['connections'] as const,
  lists: () => [...connectionKeys.all, 'list'] as const,
  list: (provider?: string) => [...connectionKeys.lists(), { provider }] as const,
  detail: (id: string) => [...connectionKeys.all, 'detail', id] as const,
  models: (connectionId: string) => [...connectionKeys.all, 'models', connectionId] as const,
  defaultConnection: () => [...connectionKeys.all, 'default'] as const,
};

/**
 * Hook to get all connections for the current user
 */
export const useConnections = (provider?: string) => {
  return useQuery({
    queryKey: connectionKeys.list(provider),
    queryFn: async () => {
      const params = provider ? `?provider=${provider}` : '';
      const response = await apiClient.get<{ connections: Connection[] }>(`/connections${params}`);
      // Handle both response formats (direct array or wrapped in connections property)
      if (Array.isArray(response)) {
        return response;
      }
      return response.connections || [];
    },
    staleTime: 30 * 1000, // Consider fresh for 30 seconds
  });
};

/**
 * Hook to get a single connection
 */
export const useConnection = (id?: string) => {
  return useQuery({
    queryKey: connectionKeys.detail(id!),
    queryFn: async () => {
      return apiClient.get<Connection>(`/connections/${id}`);
    },
    enabled: !!id,
  });
};

/**
 * Hook to get the default connection
 */
export const useDefaultConnection = () => {
  return useQuery({
    queryKey: connectionKeys.defaultConnection(),
    queryFn: async () => {
      try {
        return await apiClient.get<Connection>('/connections/default');
      } catch (error) {
        // If no default connection, return null
        return null;
      }
    },
    staleTime: 30 * 1000,
  });
};

/**
 * Hook to get available models for a connection
 */
export const useConnectionModels = (connectionId?: string) => {
  return useQuery({
    queryKey: connectionKeys.models(connectionId!),
    queryFn: async () => {
      return apiClient.get<Model[]>(`/connections/${connectionId}/models`);
    },
    enabled: !!connectionId,
    staleTime: 5 * 60 * 1000, // Consider fresh for 5 minutes
  });
};

/**
 * Hook to create a new connection
 */
export const useCreateConnection = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: CreateConnectionRequest) => {
      return apiClient.post<Connection>('/connections', data);
    },
    onSuccess: (newConnection) => {
      // If this is set as default, update other connections
      if (newConnection.is_default) {
        queryClient.setQueryData<Connection[]>(
          connectionKeys.list(),
          (old) => old?.map(conn => ({ ...conn, is_default: false })) || []
        );
      }
      
      // Add new connection to cache
      queryClient.setQueryData<Connection[]>(
        connectionKeys.list(),
        (old) => [...(old || []), newConnection]
      );
      
      // Invalidate queries
      queryClient.invalidateQueries({ queryKey: connectionKeys.lists() });
      
      if (newConnection.is_default) {
        queryClient.setQueryData(connectionKeys.defaultConnection(), newConnection);
      }
    },
  });
};

/**
 * Hook to update a connection
 */
export const useUpdateConnection = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, ...data }: UpdateConnectionRequest) => {
      return apiClient.put<Connection>(`/connections/${id}`, data);
    },
    onSuccess: (updatedConnection) => {
      // If this is set as default, update other connections
      if (updatedConnection.is_default) {
        queryClient.setQueryData<Connection[]>(
          connectionKeys.list(),
          (old) => old?.map(conn => ({
            ...conn,
            is_default: conn.id === updatedConnection.id
          })) || []
        );
        queryClient.setQueryData(connectionKeys.defaultConnection(), updatedConnection);
      }
      
      // Update specific connection in cache
      queryClient.setQueryData(connectionKeys.detail(updatedConnection.id), updatedConnection);
      
      // Update in list cache
      queryClient.setQueryData<Connection[]>(
        connectionKeys.list(),
        (old) => old?.map(conn => 
          conn.id === updatedConnection.id ? updatedConnection : conn
        ) || []
      );
      
      // Invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: connectionKeys.lists() });
    },
  });
};

/**
 * Hook to delete a connection
 */
export const useDeleteConnection = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      return apiClient.delete(`/connections/${id}`);
    },
    onSuccess: (_, deletedId) => {
      // Remove from cache
      queryClient.setQueryData<Connection[]>(
        connectionKeys.list(),
        (old) => old?.filter(conn => conn.id !== deletedId) || []
      );
      
      // Invalidate queries
      queryClient.invalidateQueries({ queryKey: connectionKeys.lists() });
      queryClient.removeQueries({ queryKey: connectionKeys.detail(deletedId) });
      queryClient.removeQueries({ queryKey: connectionKeys.models(deletedId) });
    },
  });
};

/**
 * Hook to test a connection
 */
export const useTestConnection = () => {
  return useMutation({
    mutationFn: async (id: string) => {
      return apiClient.post<{
        success: boolean;
        models?: Model[];
        error?: string;
        latency_ms?: number;
      }>(`/connections/${id}/test`);
    },
  });
};

/**
 * Hook to refresh models for a connection
 */
export const useRefreshModels = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (connectionId: string) => {
      return apiClient.post<Model[]>(`/connections/${connectionId}/refresh-models`);
    },
    onSuccess: (models, connectionId) => {
      // Update models cache
      queryClient.setQueryData(connectionKeys.models(connectionId), models);
    },
  });
};

/**
 * Hook to set a connection as default
 */
export const useSetDefaultConnection = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      return apiClient.put<Connection>(`/connections/${id}/set-default`);
    },
    onSuccess: (updatedConnection) => {
      // Update all connections in cache
      queryClient.setQueryData<Connection[]>(
        connectionKeys.list(),
        (old) => old?.map(conn => ({
          ...conn,
          is_default: conn.id === updatedConnection.id
        })) || []
      );
      
      // Update default connection cache
      queryClient.setQueryData(connectionKeys.defaultConnection(), updatedConnection);
      
      // Invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: connectionKeys.lists() });
    },
  });
};