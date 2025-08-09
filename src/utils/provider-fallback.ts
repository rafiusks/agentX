/**
 * Provider fallback utilities for automatic failover
 */

import { useConnectionStore } from '../stores/connection.store';
import { ConnectionService } from '../services/connection.service';

export interface FallbackProvider {
  id: string;
  name: string;
  priority: number;
}

/**
 * Get ordered list of available providers for fallback
 */
export async function getAvailableProviders(): Promise<FallbackProvider[]> {
  const connectionService = new ConnectionService();
  const connections = await connectionService.getConnections();
  
  // Filter and sort by priority
  return connections
    .filter(conn => conn.status === 'connected')
    .map(conn => ({
      id: conn.id,
      name: conn.name,
      priority: getProviderPriority(conn.provider)
    }))
    .sort((a, b) => a.priority - b.priority);
}

/**
 * Get provider priority (lower is better)
 */
function getProviderPriority(provider: string): number {
  const priorities: Record<string, number> = {
    'openai': 1,      // Fastest, most reliable
    'anthropic': 2,   // Very reliable, good for complex tasks
    'groq': 3,        // Fast but less reliable
    'ollama': 4,      // Local, depends on user's hardware
    'demo': 99        // Last resort
  };
  
  return priorities[provider.toLowerCase()] || 50;
}

/**
 * Get next fallback provider
 */
export async function getNextFallbackProvider(
  currentProviderId: string
): Promise<string | null> {
  const providers = await getAvailableProviders();
  const currentIndex = providers.findIndex(p => p.id === currentProviderId);
  
  if (currentIndex === -1 || currentIndex === providers.length - 1) {
    return null; // No fallback available
  }
  
  return providers[currentIndex + 1].id;
}

/**
 * Attempt request with automatic provider fallback
 */
export async function withProviderFallback<T>(
  request: (connectionId: string) => Promise<T>,
  initialConnectionId: string,
  onFallback?: (from: string, to: string) => void
): Promise<T> {
  const providers = await getAvailableProviders();
  let currentProviderId = initialConnectionId;
  let lastError: any;
  
  // Find starting index
  let startIndex = providers.findIndex(p => p.id === currentProviderId);
  if (startIndex === -1) startIndex = 0;
  
  // Try each provider in order
  for (let i = startIndex; i < providers.length; i++) {
    const provider = providers[i];
    
    try {
      return await request(provider.id);
    } catch (error: any) {
      lastError = error;
      
      // Check if error is recoverable
      if (!isRecoverableError(error)) {
        throw error; // Don't fallback for non-recoverable errors
      }
      
      // Try next provider
      const nextProvider = providers[i + 1];
      if (nextProvider) {
        console.log(`[Fallback] Switching from ${provider.name} to ${nextProvider.name}`);
        onFallback?.(provider.name, nextProvider.name);
        currentProviderId = nextProvider.id;
      }
    }
  }
  
  throw new Error(`All providers failed. Last error: ${lastError?.message || 'Unknown'}`);
}

/**
 * Check if error is recoverable with fallback
 */
function isRecoverableError(error: any): boolean {
  // Rate limits are recoverable
  if (error.status === 429) return true;
  
  // Server errors are recoverable
  if (error.status >= 500) return true;
  
  // Network errors are recoverable
  if (error.code === 'ECONNREFUSED') return true;
  if (error.code === 'ETIMEDOUT') return true;
  
  // Model not found might be recoverable with different provider
  if (error.message?.includes('model not found')) return true;
  
  // Authentication errors are not recoverable
  if (error.status === 401 || error.status === 403) return false;
  
  // Bad request is not recoverable
  if (error.status === 400) return false;
  
  // Default to recoverable
  return true;
}