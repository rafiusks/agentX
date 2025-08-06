// API service for communicating with the Go backend
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

export interface ProviderInfo {
  id: string;
  name: string;
  enabled: boolean;
  status: string;
  type: string;
  models?: ModelInfo[];
}

export interface ModelInfo {
  id: string;
  provider: string;
  display_name: string;
  description: string;
  capabilities: {
    chat: boolean;
    streaming: boolean;
    function_calling: boolean;
    vision: boolean;
  };
  status: {
    available: boolean;
    health: string;
  };
}

export interface ChatRequest {
  messages: Message[];
  preferences?: {
    speed?: 'fast' | 'balanced' | 'quality';
    cost?: 'economy' | 'standard' | 'premium';
    privacy?: 'local' | 'cloud';
    provider?: string;
    model?: string;
  };
  session_id?: string;
  temperature?: number;
  max_tokens?: number;
}

export interface Message {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface ChatResponse {
  id: string;
  content: string;
  role: string;
  metadata: {
    provider: string;
    model: string;
    latency_ms: number;
    routing_reason?: string;
  };
  usage: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
    estimated_cost?: number;
  };
}

export interface Session {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
  provider?: string;
  model?: string;
}

class ApiService {
  private getAuthHeaders(): Record<string, string> {
    const token = localStorage.getItem('access_token');
    if (token) {
      return { Authorization: `Bearer ${token}` };
    }
    return {};
  }

  private async fetch(path: string, options?: RequestInit) {
    const response = await fetch(`${API_BASE_URL}${path}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...this.getAuthHeaders(),
        ...options?.headers,
      },
    });

    if (!response.ok) {
      // Handle 401 Unauthorized - token might be expired
      if (response.status === 401) {
        // The auth interceptor in auth.ts will handle token refresh
        // For now, just throw the error
        const error = await response.json().catch(() => ({ error: 'Unauthorized' }));
        throw new Error(error.error || 'Authentication required');
      }
      
      const error = await response.json().catch(() => ({ error: response.statusText }));
      throw new Error(error.error || `API request failed: ${response.status}`);
    }

    return response.json();
  }

  // Provider methods
  async getProviders(): Promise<ProviderInfo[]> {
    return this.fetch('/providers');
  }

  async updateProviderConfig(providerId: string, config: any) {
    return this.fetch(`/providers/${providerId}/config`, {
      method: 'PUT',
      body: JSON.stringify(config),
    });
  }

  async discoverModels(providerId: string) {
    return this.fetch(`/providers/${providerId}/discover`, {
      method: 'POST',
    });
  }

  async getProvidersHealth() {
    return this.fetch('/providers/health');
  }

  // Chat methods
  async chat(request: ChatRequest): Promise<ChatResponse> {
    return this.fetch('/chat', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async streamChat(request: ChatRequest, onChunk: (chunk: any) => void) {
    const response = await fetch(`${API_BASE_URL}/chat/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...this.getAuthHeaders(),
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`Stream request failed: ${response.status}`);
    }

    const reader = response.body?.getReader();
    const decoder = new TextDecoder();

    if (!reader) {
      throw new Error('No response body');
    }

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      const chunk = decoder.decode(value);
      const lines = chunk.split('\n');

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6);
          if (data === '[DONE]') continue;
          
          try {
            const parsed = JSON.parse(data);
            onChunk(parsed);
          } catch (e) {
            console.error('Failed to parse chunk:', e);
          }
        }
      }
    }
  }

  // Model methods
  async getModels(): Promise<ModelInfo[]> {
    const response = await this.fetch('/models');
    return response.models || [];
  }

  // Session methods
  async createSession(title: string): Promise<Session> {
    return this.fetch('/sessions', {
      method: 'POST',
      body: JSON.stringify({ title }),
    });
  }

  async getSessions(): Promise<Session[]> {
    const response = await this.fetch('/sessions');
    return response.sessions || [];
  }

  async getSession(id: string): Promise<Session> {
    return this.fetch(`/sessions/${id}`);
  }

  async deleteSession(id: string) {
    return this.fetch(`/sessions/${id}`, {
      method: 'DELETE',
    });
  }

  async getSessionMessages(sessionId: string): Promise<Message[]> {
    const response = await this.fetch(`/sessions/${sessionId}/messages`);
    return response.messages || [];
  }

  // Settings methods
  async getSettings() {
    return this.fetch('/settings');
  }

  async updateSettings(settings: any) {
    return this.fetch('/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    });
  }

  // Connection methods
  async listConnections(providerId?: string) {
    const query = providerId ? `?provider_id=${providerId}` : '';
    const response = await this.fetch(`/connections${query}`);
    console.log('listConnections response:', response);
    return response.connections || [];
  }

  async getConnection(id: string) {
    return this.fetch(`/connections/${id}`);
  }

  async createConnection(data: {
    provider_id: string;
    name: string;
    config: any;
  }) {
    return this.fetch('/connections', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateConnection(id: string, updates: any) {
    return this.fetch(`/connections/${id}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  async deleteConnection(id: string) {
    return this.fetch(`/connections/${id}`, {
      method: 'DELETE',
    });
  }

  async toggleConnection(id: string) {
    return this.fetch(`/connections/${id}/toggle`, {
      method: 'POST',
    });
  }

  async testConnection(id: string) {
    return this.fetch(`/connections/${id}/test`, {
      method: 'POST',
    });
  }

  async setDefaultConnection(id: string) {
    return this.fetch(`/connections/${id}/set-default`, {
      method: 'POST',
    });
  }
}

export const api = new ApiService();