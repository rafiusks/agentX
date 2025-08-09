import { QueryClient } from '@tanstack/react-query';
import { ApiError } from './api-client';

/**
 * Configure TanStack Query client with smart defaults
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Consider data fresh for 1 minute
      staleTime: 1000 * 60,
      // Keep cache for 5 minutes
      gcTime: 1000 * 60 * 5,
      // Retry logic
      retry: (failureCount, error) => {
        // Don't retry on 4xx errors (client errors)
        if (error instanceof ApiError) {
          if (error.status >= 400 && error.status < 500) {
            return false;
          }
        }
        // Retry up to 3 times for other errors
        return failureCount < 3;
      },
      // Retry delay with exponential backoff
      retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
      // Refetch on window focus by default
      refetchOnWindowFocus: true,
      // Don't refetch on reconnect by default (we have retry logic)
      refetchOnReconnect: 'always',
    },
    mutations: {
      // Retry failed mutations once
      retry: 1,
      retryDelay: 1000,
    },
  },
});

// Global error handler for auth issues
queryClient.setMutationDefaults(['auth'], {
  mutationFn: async (_variables: unknown) => {
    throw new Error('Mutation function not implemented');
  },
  onError: (error) => {
    if (error instanceof ApiError && error.status === 401) {
      // Clear auth data and redirect to login
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      queryClient.clear();
      window.location.href = '/login';
    }
  },
});

// Helper to invalidate queries
export const invalidateQueries = (queryKey: string[]) => {
  return queryClient.invalidateQueries({ queryKey });
};

// Helper to prefetch queries
export const prefetchQuery = (queryKey: string[], queryFn: () => Promise<unknown>) => {
  return queryClient.prefetchQuery({ queryKey, queryFn });
};

// Helper to get cached data
export const getCachedData = <T = unknown>(queryKey: string[]): T | undefined => {
  return queryClient.getQueryData<T>(queryKey);
};

// Helper to set cached data
export const setCachedData = <T = unknown>(queryKey: string[], data: T) => {
  queryClient.setQueryData(queryKey, data);
};