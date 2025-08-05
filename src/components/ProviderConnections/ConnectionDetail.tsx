import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useToast } from '@/components/ui/use-toast';
import { Loader2, Save, Trash2, TestTube, Check, X, Star } from 'lucide-react';
import { api } from '@/services/api';

interface ConnectionDetailProps {
  connectionId: string;
  onUpdate: () => void;
  onDelete: () => void;
}

interface ConnectionData {
  id: string;
  provider_id: string;
  name: string;
  enabled: boolean;
  config: {
    api_key?: string;
    base_url?: string;
    models?: string[];
    default_model?: string;
  };
  metadata?: {
    last_tested?: string;
    test_status?: string;
    test_message?: string;
  };
  status: string;
}

export const ConnectionDetail: React.FC<ConnectionDetailProps> = ({
  connectionId,
  onUpdate,
  onDelete,
}) => {
  const [connection, setConnection] = useState<ConnectionData | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const { toast } = useToast();

  useEffect(() => {
    loadConnection();
  }, [connectionId]);

  const loadConnection = async () => {
    try {
      setLoading(true);
      const data = await api.getConnection(connectionId);
      setConnection(data);
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to load connection details',
        variant: 'destructive',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!connection) return;

    setSaving(true);
    try {
      await api.updateConnection(connectionId, {
        name: connection.name,
        config: connection.config,
      });
      
      toast({
        title: 'Success',
        description: 'Connection saved successfully',
      });
      
      onUpdate();
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to save connection',
        variant: 'destructive',
      });
    } finally {
      setSaving(false);
    }
  };

  const handleToggle = async () => {
    if (!connection) return;

    try {
      const updated = await api.toggleConnection(connectionId);
      setConnection(updated);
      onUpdate();
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to toggle connection',
        variant: 'destructive',
      });
    }
  };

  const handleTest = async () => {
    setTesting(true);
    try {
      const result = await api.testConnection(connectionId);
      
      toast({
        title: result.success ? 'Success' : 'Failed',
        description: result.message,
        variant: result.success ? 'default' : 'destructive',
      });
      
      // Reload to get updated metadata
      await loadConnection();
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to test connection',
        variant: 'destructive',
      });
    } finally {
      setTesting(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this connection?')) return;

    setDeleting(true);
    try {
      await api.deleteConnection(connectionId);
      
      toast({
        title: 'Success',
        description: 'Connection deleted successfully',
      });
      
      onDelete();
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to delete connection',
        variant: 'destructive',
      });
    } finally {
      setDeleting(false);
    }
  };

  const handleSetDefault = async () => {
    try {
      await api.setDefaultConnection(connectionId);
      
      toast({
        title: 'Success',
        description: 'Default connection set successfully',
      });
      
      onUpdate();
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to set default connection',
        variant: 'destructive',
      });
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  if (!connection) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-foreground-secondary">Connection not found</p>
      </div>
    );
  }

  const providerLabels: Record<string, string> = {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    ollama: 'Ollama',
    'lm-studio': 'LM Studio',
  };

  return (
    <div className="p-6 space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-foreground-primary">{connection.name}</CardTitle>
              <CardDescription className="text-foreground-secondary">
                {providerLabels[connection.provider_id]} Connection
              </CardDescription>
            </div>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleSetDefault}
              >
                <Star className="h-4 w-4 mr-2" />
                Set as Default
              </Button>
              <Switch
                checked={connection.enabled}
                onCheckedChange={handleToggle}
              />
            </div>
          </div>
        </CardHeader>
        
        <CardContent className="space-y-4">
          {/* Connection Name */}
          <div>
            <Label htmlFor="name" className="text-foreground-primary">Connection Name</Label>
            <Input
              id="name"
              value={connection.name}
              onChange={(e) => setConnection({ ...connection, name: e.target.value })}
              placeholder="e.g., Work API, Personal Project"
              className="bg-background-primary border-border-subtle text-foreground-primary placeholder:text-foreground-tertiary"
            />
          </div>

          {/* API Key */}
          {connection.provider_id !== 'ollama' && (
            <div>
              <Label htmlFor="api-key" className="text-foreground-primary">API Key</Label>
              <Input
                id="api-key"
                type="password"
                value={connection.config.api_key || ''}
                onChange={(e) => setConnection({
                  ...connection,
                  config: { ...connection.config, api_key: e.target.value }
                })}
                placeholder={connection.provider_id === 'openai' ? 'sk-...' : 'Your API key'}
                className="bg-background-primary border-border-subtle text-foreground-primary placeholder:text-foreground-tertiary"
              />
            </div>
          )}

          {/* Base URL for compatible providers */}
          {(connection.provider_id === 'ollama' || connection.provider_id === 'lm-studio') && (
            <div>
              <Label htmlFor="base-url" className="text-foreground-primary">Base URL</Label>
              <Input
                id="base-url"
                value={connection.config.base_url || ''}
                onChange={(e) => setConnection({
                  ...connection,
                  config: { ...connection.config, base_url: e.target.value }
                })}
                placeholder="http://localhost:11434"
                className="bg-background-primary border-border-subtle text-foreground-primary placeholder:text-foreground-tertiary"
              />
            </div>
          )}

          {/* Connection Status */}
          <div className="p-4 bg-background-tertiary rounded-lg border border-border-subtle">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-foreground-primary">Connection Status</p>
                <p className="text-sm text-foreground-secondary">
                  {connection.status === 'connected' && (
                    <span className="flex items-center text-accent-green">
                      <Check className="h-4 w-4 mr-1" />
                      Connected
                    </span>
                  )}
                  {connection.status === 'error' && (
                    <span className="flex items-center text-accent-red">
                      <X className="h-4 w-4 mr-1" />
                      Error
                    </span>
                  )}
                  {connection.status === 'disconnected' && (
                    <span className="text-foreground-secondary">Disconnected</span>
                  )}
                  {connection.status === 'disabled' && (
                    <span className="text-foreground-tertiary">Disabled</span>
                  )}
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleTest}
                disabled={testing || !connection.enabled}
              >
                {testing ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <TestTube className="h-4 w-4 mr-2" />
                )}
                Test Connection
              </Button>
            </div>
            
            {connection.metadata?.last_tested && (
              <p className="text-xs text-foreground-tertiary mt-2">
                Last tested: {new Date(connection.metadata.last_tested).toLocaleString()}
              </p>
            )}
          </div>

          {/* Action Buttons */}
          <div className="flex justify-between pt-4">
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleting}
            >
              {deleting ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Trash2 className="h-4 w-4 mr-2" />
              )}
              Delete
            </Button>
            
            <Button
              onClick={handleSave}
              disabled={saving}
            >
              {saving ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Save className="h-4 w-4 mr-2" />
              )}
              Save Changes
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};