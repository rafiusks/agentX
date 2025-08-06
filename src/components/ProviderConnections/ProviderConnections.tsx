import React, { useState, useEffect } from 'react';
import { ConnectionSidebar, type ProviderGroup } from './ConnectionSidebar';
import { ConnectionDetail } from './ConnectionDetail';
import { AddConnectionModal } from './AddConnectionModal';
import { EmptyState } from './EmptyState';
import { api } from '@/services/api';
import { useToast } from '@/components/ui/use-toast';

interface ProviderConnectionsProps {
  onClose?: () => void;
}

export const ProviderConnections: React.FC<ProviderConnectionsProps> = ({ onClose }) => {
  const [providers, setProviders] = useState<ProviderGroup[]>([]);
  const [selectedConnectionId, setSelectedConnectionId] = useState<string | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);
  const [addingToProvider, setAddingToProvider] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const { toast } = useToast();

  useEffect(() => {
    loadConnections();
  }, []);

  const loadConnections = async () => {
    try {
      setLoading(true);
      const connections = await api.listConnections();
      
      // Group connections by provider
      const grouped = connections.reduce((acc, conn) => {
        // Map openai-compatible connections to lm-studio if they use port 1234
        let groupKey = conn.provider_id;
        if (conn.provider_id === 'openai-compatible' && 
            conn.config?.base_url?.includes('localhost:1234')) {
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

      setProviders(providerGroups);
      
      // Select first connection if none selected
      if (!selectedConnectionId && connections.length > 0) {
        setSelectedConnectionId(connections[0].id);
      }
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to load connections',
        variant: 'destructive',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleAddConnection = (providerId: string) => {
    setAddingToProvider(providerId);
    setShowAddModal(true);
  };

  const handleConnectionCreated = (connectionId: string) => {
    setShowAddModal(false);
    setAddingToProvider(null);
    loadConnections();
    setSelectedConnectionId(connectionId);
  };

  const handleConnectionDeleted = () => {
    setSelectedConnectionId(null);
    loadConnections();
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900"></div>
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
            onUpdate={loadConnections}
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