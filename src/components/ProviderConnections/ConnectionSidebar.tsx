import React from 'react';
import { ChevronRight, Plus, Circle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export interface Connection {
  id: string;
  provider_id: string;
  name: string;
  enabled: boolean;
  status: 'connected' | 'connecting' | 'error' | 'disconnected' | 'disabled';
}

export interface ProviderGroup {
  providerId: string;
  providerName: string;
  providerIcon: string;
  connections: Connection[];
}

interface ConnectionSidebarProps {
  providers: ProviderGroup[];
  selectedConnectionId?: string;
  onConnectionSelect: (connectionId: string) => void;
  onAddConnection: (providerId: string) => void;
}

const providerIcons: Record<string, string> = {
  openai: '⚡',
  anthropic: '🧠',
  ollama: '🦙',
  'lm-studio': '🖥️',
};

const statusColors: Record<string, string> = {
  connected: 'text-accent-green',
  connecting: 'text-accent-yellow',
  error: 'text-accent-red',
  disconnected: 'text-foreground-muted',
  disabled: 'text-foreground-tertiary',
};

export const ConnectionSidebar: React.FC<ConnectionSidebarProps> = ({
  providers,
  selectedConnectionId,
  onConnectionSelect,
  onAddConnection,
}) => {
  const [expandedProviders, setExpandedProviders] = React.useState<Set<string>>(
    new Set(providers.map(p => p.providerId))
  );

  const toggleProvider = (providerId: string) => {
    const newExpanded = new Set(expandedProviders);
    if (newExpanded.has(providerId)) {
      newExpanded.delete(providerId);
    } else {
      newExpanded.add(providerId);
    }
    setExpandedProviders(newExpanded);
  };

  const hasConnections = providers.some(p => p.connections.length > 0);

  return (
    <div className="w-64 h-full border-r border-border-subtle bg-background-secondary">
      <div className="p-4">
        <h2 className="text-lg font-semibold mb-4 text-foreground-primary">Connections</h2>
        
        {hasConnections ? (
          <div className="space-y-2">
            {providers.map((provider) => {
              if (provider.connections.length === 0) return null;
              
              return (
                <div key={provider.providerId} className="space-y-1">
                  {/* Provider Header */}
                  <div
                    className="flex items-center justify-between p-2 rounded hover:bg-background-tertiary cursor-pointer"
                    onClick={() => toggleProvider(provider.providerId)}
                  >
                    <div className="flex items-center space-x-2">
                      <ChevronRight
                        className={cn(
                          "h-4 w-4 transition-transform text-foreground-secondary",
                          expandedProviders.has(provider.providerId) && "rotate-90"
                        )}
                      />
                      <span className="text-lg">{providerIcons[provider.providerId] || '📡'}</span>
                      <span className="font-medium text-foreground-primary">{provider.providerName}</span>
                      <span className="text-xs text-foreground-muted">({provider.connections.length})</span>
                    </div>
                  </div>

                  {/* Connections List */}
                  {expandedProviders.has(provider.providerId) && (
                    <div className="ml-6 space-y-1">
                      {provider.connections.map((connection) => (
                        <div
                          key={connection.id}
                          className={cn(
                            "flex items-center justify-between p-2 rounded cursor-pointer",
                            "hover:bg-background-tertiary",
                            selectedConnectionId === connection.id && "bg-accent-blue/10"
                          )}
                          onClick={() => onConnectionSelect(connection.id)}
                        >
                          <div className="flex items-center space-x-2">
                            <Circle
                              className={cn("h-2 w-2 fill-current", statusColors[connection.status])}
                            />
                            <span className="text-sm text-foreground-primary">{connection.name}</span>
                            {connection.enabled && (
                              <span className="text-accent-green">✓</span>
                            )}
                          </div>
                        </div>
                      ))}
                      
                      {/* Add Connection Button */}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="w-full justify-start"
                        onClick={() => onAddConnection(provider.providerId)}
                      >
                        <Plus className="h-4 w-4 mr-2" />
                        Add Connection
                      </Button>
                    </div>
                  )}
                </div>
              );
            })}
            
            {/* New Connection Button */}
            <div className="mt-4 pt-4 border-t border-border-subtle">
              <Button
                variant="outline"
                size="sm"
                className="w-full"
                onClick={() => onAddConnection('openai')}
              >
                <Plus className="h-4 w-4 mr-2" />
                New Connection
              </Button>
            </div>
          </div>
        ) : (
          <div className="text-center py-8">
            <p className="text-sm text-foreground-muted mb-4">
              No connections configured yet
            </p>
            <Button
              variant="outline"
              size="sm"
              className="w-full"
              onClick={() => onAddConnection('openai')}
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Connection
            </Button>
          </div>
        )}
      </div>
    </div>
  );
};