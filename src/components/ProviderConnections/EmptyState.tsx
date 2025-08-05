import React from 'react';
import { Button } from '@/components/ui/button';
import { Plus, Zap } from 'lucide-react';

interface EmptyStateProps {
  hasConnections: boolean;
  onAddConnection: () => void;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  hasConnections,
  onAddConnection,
}) => {
  return (
    <div className="flex flex-col items-center justify-center h-full p-8">
      <div className="text-center max-w-md">
        {hasConnections ? (
          <>
            <Zap className="h-12 w-12 text-foreground-tertiary mx-auto mb-4" />
            <h3 className="text-lg font-semibold mb-2 text-foreground-primary">Select a Connection</h3>
            <p className="text-foreground-secondary mb-6">
              Choose a connection from the sidebar to view and manage its settings
            </p>
            <p className="text-sm text-foreground-muted">
              Or create a new connection to get started with a different API key or configuration
            </p>
          </>
        ) : (
          <>
            <div className="rounded-full bg-background-tertiary p-4 inline-flex mb-4">
              <Zap className="h-8 w-8 text-foreground-secondary" />
            </div>
            <h3 className="text-lg font-semibold mb-2 text-foreground-primary">No Connections Yet</h3>
            <p className="text-foreground-secondary mb-6">
              Create your first connection to start using AI providers
            </p>
            <Button onClick={onAddConnection} className="btn-primary">
              <Plus className="h-4 w-4 mr-2" />
              Add Your First Connection
            </Button>
          </>
        )}
      </div>
    </div>
  );
};