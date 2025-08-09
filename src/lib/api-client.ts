/**
 * API Client using native fetch
 * Replaces axios with a lightweight fetch wrapper
 */

import type { ApiErrorResponse } from '@/types/api.types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public data?: ApiErrorResponse
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

interface RequestConfig extends RequestInit {
  params?: Record<string, string>;
}

class ApiClient {
  private baseURL: string;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;
  }

  private async request<T = unknown>(
    endpoint: string,
    config: RequestConfig = {}
  ): Promise<T> {
    const { params, headers, ...fetchConfig } = config;
    
    // Build URL with query params
    const url = new URL(`${this.baseURL}${endpoint}`);
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        url.searchParams.append(key, value);
      });
    }

    // Get token from localStorage
    const token = localStorage.getItem('access_token');

    // Make the request
    const response = await fetch(url.toString(), {
      ...fetchConfig,
      credentials: 'include', // Always include cookies
      headers: {
        'Content-Type': 'application/json',
        ...(token && { 'Authorization': `Bearer ${token}` }),
        ...headers,
      },
    });

    // Handle response
    if (!response.ok) {
      let errorData;
      try {
        errorData = await response.json();
      } catch {
        errorData = { message: `HTTP ${response.status}: ${response.statusText}` };
      }
      
      throw new ApiError(
        errorData.error || errorData.message || 'Request failed',
        response.status,
        errorData
      );
    }

    // Return JSON response or empty object for 204 No Content
    if (response.status === 204) {
      return {} as T;
    }

    try {
      return await response.json();
    } catch {
      return {} as T;
    }
  }

  // HTTP method helpers
  async get<T = unknown>(endpoint: string, params?: Record<string, string>): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET', params });
  }

  async post<T = unknown>(endpoint: string, data?: unknown, config?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, {
      ...config,
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T = unknown>(endpoint: string, data?: unknown, config?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, {
      ...config,
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async patch<T = unknown>(endpoint: string, data?: unknown, config?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, {
      ...config,
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T = unknown>(endpoint: string, config?: RequestConfig): Promise<T> {
    return this.request<T>(endpoint, {
      ...config,
      method: 'DELETE',
    });
  }

  // Special method for streaming responses (SSE)
  async stream(
    endpoint: string,
    data?: unknown,
    onMessage?: (message: string) => void,
    onError?: (error: Error) => void
  ): Promise<void> {
    console.log('[ApiClient.stream] Starting stream to:', endpoint, 'with data:', data);
    const token = localStorage.getItem('access_token');
    
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...(token && { 'Authorization': `Bearer ${token}` }),
      },
      body: data ? JSON.stringify(data) : undefined,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Stream request failed' }));
      console.error('[ApiClient.stream] Stream request failed:', response.status, error);
      throw new ApiError(error.message || 'Stream failed', response.status, error);
    }
    
    console.log('[ApiClient.stream] Response OK, starting to read stream');

    const reader = response.body?.getReader();
    const decoder = new TextDecoder();

    if (!reader) {
      throw new Error('Response body is not readable');
    }

    try {
      let done = false;
      while (!done) {
        const result = await reader.read();
        done = result.done;
        const value = result.value;
        if (done) break;

        const chunk = decoder.decode(value, { stream: true });
        const lines = chunk.split('\n');

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6);
            if (data === '[DONE]') {
              console.log('[ApiClient.stream] Received [DONE], ending stream');
              return;
            }
            try {
              const parsed = JSON.parse(data);
              console.log('[ApiClient.stream] Parsed chunk:', parsed);
              onMessage?.(parsed);
            } catch (e) {
              // If not JSON, pass raw data
              console.log('[ApiClient.stream] Raw data chunk:', data);
              onMessage?.(data);
            }
          }
        }
      }
    } catch (error) {
      onError?.(error as Error);
      throw error;
    } finally {
      reader.releaseLock();
    }
  }
}

// Export singleton instance
export const apiClient = new ApiClient();

// Export class for testing or custom instances
export { ApiClient };