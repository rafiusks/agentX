// AgentX API Client - HTTP/WebSocket communication with backend

export interface Provider {
  id: string;
  name: string;
  type: string;
  models: string[];
  base_url?: string;
}

export interface Session {
  id: string;
  title: string;
  messages: Message[];
  created_at: string;
  updated_at: string;
  provider?: string;
  model?: string;
}

export interface Message {
  role: 'user' | 'assistant' | 'system' | 'tool';
  content: string;
  function_call?: FunctionCall;
  tool_calls?: ToolCall[];
  tool_call_id?: string;
}

export interface FunctionCall {
  name: string;
  arguments: string;
}

export interface ToolCall {
  id: string;
  type: string;
  function: FunctionCall;
}

export interface StreamChunk {
  id?: string;
  object?: string;
  created?: number;
  model?: string;
  delta?: string;
  role?: string;
  function_call?: FunctionCall;
  tool_calls?: ToolCall[];
  finish_reason?: string;
  error?: string;
}

export interface CompletionResponse {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: Choice[];
  usage: Usage;
}

export interface Choice {
  index: number;
  message: Message;
  finish_reason: string;
}

export interface Usage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

export interface Model {
  id: string;
  object: string;
  created?: number;
  owned_by?: string;
}

export interface Settings {
  providers: Record<string, any>;
  default_provider: string;
  default_model: string;
}

export class AgentXAPI {
  private baseURL: string;
  private ws: WebSocket | null = null;

  constructor(baseURL: string = 'http://localhost:8080/api/v1') {
    this.baseURL = baseURL;
  }

  // Provider Management
  async getProviders(): Promise<Provider[]> {
    const response = await fetch(`${this.baseURL}/providers`);
    if (!response.ok) throw new Error('Failed to fetch providers');
    return response.json();
  }

  async updateProviderConfig(providerId: string, config: any): Promise<void> {
    const response = await fetch(`${this.baseURL}/providers/${providerId}/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    });
    if (!response.ok) throw new Error('Failed to update provider config');
  }

  async discoverModels(providerId: string): Promise<Model[]> {
    const response = await fetch(`${this.baseURL}/providers/${providerId}/discover`, {
      method: 'POST',
    });
    if (!response.ok) throw new Error('Failed to discover models');
    return response.json();
  }

  // Session Management
  async createSession(title?: string): Promise<Session> {
    const response = await fetch(`${this.baseURL}/chat/sessions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: title || 'New Chat' }),
    });
    if (!response.ok) throw new Error('Failed to create session');
    return response.json();
  }

  async getSessions(): Promise<Session[]> {
    const response = await fetch(`${this.baseURL}/chat/sessions`);
    if (!response.ok) throw new Error('Failed to fetch sessions');
    return response.json();
  }

  async getSession(sessionId: string): Promise<Session> {
    const response = await fetch(`${this.baseURL}/chat/sessions/${sessionId}`);
    if (!response.ok) throw new Error('Failed to fetch session');
    return response.json();
  }

  async deleteSession(sessionId: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/chat/sessions/${sessionId}`, {
      method: 'DELETE',
    });
    if (!response.ok) throw new Error('Failed to delete session');
  }

  // Non-streaming message
  async sendMessage(
    sessionId: string,
    message: string,
    providerId: string,
    model: string,
    options?: {
      temperature?: number;
      max_tokens?: number;
      functions?: any[];
      tools?: any[];
    }
  ): Promise<CompletionResponse> {
    const response = await fetch(`${this.baseURL}/chat/sessions/${sessionId}/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        message,
        provider_id: providerId,
        model,
        ...options,
      }),
    });
    if (!response.ok) throw new Error('Failed to send message');
    return response.json();
  }

  // Streaming message via WebSocket
  streamMessage(
    sessionId: string,
    message: string,
    providerId: string,
    model: string,
    options?: {
      temperature?: number;
      max_tokens?: number;
      functions?: any[];
      tools?: any[];
    },
    callbacks?: {
      onChunk?: (chunk: StreamChunk) => void;
      onComplete?: () => void;
      onError?: (error: string) => void;
    }
  ): () => void {
    // Close existing connection if any
    if (this.ws) {
      this.ws.close();
    }

    const wsURL = this.baseURL.replace('http://', 'ws://').replace('https://', 'wss://').replace('/api/v1', '');
    this.ws = new WebSocket(`${wsURL}/ws/chat`);

    this.ws.onopen = () => {
      this.ws?.send(JSON.stringify({
        session_id: sessionId,
        message,
        provider_id: providerId,
        model,
        ...options,
      }));
    };

    this.ws.onmessage = (event) => {
      try {
        const chunk: StreamChunk = JSON.parse(event.data);
        
        if (chunk.error) {
          callbacks?.onError?.(chunk.error);
          this.ws?.close();
        } else if (chunk.finish_reason) {
          callbacks?.onComplete?.();
          this.ws?.close();
        } else {
          callbacks?.onChunk?.(chunk);
        }
      } catch (err) {
        callbacks?.onError?.(`Failed to parse response: ${err}`);
      }
    };

    this.ws.onerror = (error) => {
      callbacks?.onError?.('WebSocket error occurred');
    };

    this.ws.onclose = () => {
      this.ws = null;
    };

    // Return cleanup function
    return () => {
      if (this.ws) {
        this.ws.close();
        this.ws = null;
      }
    };
  }

  // Settings Management
  async getSettings(): Promise<Settings> {
    const response = await fetch(`${this.baseURL}/settings`);
    if (!response.ok) throw new Error('Failed to fetch settings');
    return response.json();
  }

  async updateSettings(settings: Partial<Settings>): Promise<void> {
    const response = await fetch(`${this.baseURL}/settings`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(settings),
    });
    if (!response.ok) throw new Error('Failed to update settings');
  }

  // Health Check
  async health(): Promise<{ status: string; service: string }> {
    const response = await fetch(`${this.baseURL}/health`);
    if (!response.ok) throw new Error('Service unavailable');
    return response.json();
  }
}

// Create a singleton instance
export const api = new AgentXAPI();