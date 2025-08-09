import React, { useState, useEffect } from 'react';
import { ConnectionSidebar, type ProviderGroup } from './ConnectionSidebar';
import { ConnectionDetail } from './ConnectionDetail';
import { AddConnectionModal } from './AddConnectionModal';
import { EmptyState } from './EmptyState';
import { useConnections, useDeleteConnection, useUpdateConnection } from '@/hooks/queries/useConnections';
import { useToast } from '@/components/ui/use-toast';
import { useQueryClient } from '@tanstack/react-query';

interface ProviderConnectionsProps {
  onClose?: () => void;
}

export const ProviderConnections: React.FC<ProviderConnectionsProps> = ({ onClose: _onClose }) => {
  const [selectedConnectionId, setSelectedConnectionId] = useState<string | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);
  const [addingToProvider, setAddingToProvider] = useState<string | null>(null);
  const { toast } = useToast();
  const queryClient = useQueryClient();
  
  const { data: connections = [], isLoading: loading } = useConnections();
  const deleteConnectionMutation = useDeleteConnection();
  const updateConnectionMutation = useUpdateConnection();

  // Process connections into provider groups
  const providers = React.useMemo(() => {
    if (!connections || connections.length === 0) return [];
    
    // Group connections by provider (handle both old and new API formats)
    const grouped = connections.reduce((acc: any, conn: any) => {
      // Use provider or provider_id field
      let groupKey = conn.provider || conn.provider_id;
      
      // Map openai-compatible connections to lm-studio if they use port 1234
      if (groupKey === 'openai-compatible' && 
          (conn.config?.base_url?.includes('localhost:1234') || 
           conn.base_url?.includes('localhost:1234'))) {
        groupKey = 'lm-studio';
      }
      
      if (!acc[groupKey]) {
        acc[groupKey] = [];
      }
      acc[groupKey].push(conn);
      return acc;
    }, {} as Record<string, typeof connections>);

    // Convert to provider groups
    const providerGroups: ProviderGroup[] = [
      {
        providerId: 'openai',
        providerName: 'OpenAI',
        providerIcon: '⚡',
        connections: grouped['openai'] || [],
      },
      {
        providerId: 'anthropic',
        providerName: 'Anthropic',
        providerIcon: '🧠',
        connections: grouped['anthropic'] || [],
      },
      {
        providerId: 'ollama',
        providerName: 'Ollama',
        providerIcon: '🦙',
        connections: grouped['ollama'] || [],
      },
      {
        providerId: 'lm-studio',
        providerName: 'LM Studio',
        providerIcon: '🖥️',
        connections: grouped['lm-studio'] || [],
      },
      {
        providerId: 'openai-compatible',
        providerName: 'OpenAI Compatible',
        providerIcon: '🔧',
        connections: grouped['openai-compatible'] || [],
      },
    ];

    return providerGroups;
  }, [connections]);
  
  // Select first connection if none selected
  useEffect(() => {
    if (!selectedConnectionId && connections.length > 0) {
      setSelectedConnectionId(connections[0].id);
    }
  }, [selectedConnectionId, connections]);

  const handleAddConnection = (providerId: string) => {
    setAddingToProvider(providerId);
    setShowAddModal(true);
  };

  const handleConnectionCreated = (connectionId: string) => {
    setShowAddModal(false);
    setAddingToProvider(null);
    queryClient.invalidateQueries({ queryKey: ['connections'] });
    setSelectedConnectionId(connectionId);
  };

  const handleConnectionDeleted = () => {
    setSelectedConnectionId(null);
    queryClient.invalidateQueries({ queryKey: ['connections'] });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-foreground-primary"></div>
      </div>
    );
  }

  const hasConnections = providers.some(p => p.connections.length > 0);

  return (
    <div className="flex h-full bg-background-primary">
      <ConnectionSidebar
        providers={providers}
        selectedConnectionId={selectedConnectionId || undefined}
        onConnectionSelect={setSelectedConnectionId}
        onAddConnection={handleAddConnection}
      />
      
      <div className="flex-1">
        {selectedConnectionId ? (
          <ConnectionDetail
            connectionId={selectedConnectionId}
            onUpdate={() => queryClient.invalidateQueries({ queryKey: ['connections'] })}
            onDelete={handleConnectionDeleted}
          />
        ) : (
          <EmptyState
            hasConnections={hasConnections}
            onAddConnection={() => handleAddConnection('openai')}
          />
        )}
      </div>

      {showAddModal && addingToProvider && (
        <AddConnectionModal
          providerId={addingToProvider}
          onClose={() => {
            setShowAddModal(false);
            setAddingToProvider(null);
          }}
          onSuccess={handleConnectionCreated}
        />
      )}
    </div>
  );
};