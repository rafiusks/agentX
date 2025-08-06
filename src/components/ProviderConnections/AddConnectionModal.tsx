import React, { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useToast } from '@/components/ui/use-toast';
import { Loader2 } from 'lucide-react';
import { api } from '@/services/api';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

interface AddConnectionModalProps {
  providerId: string;
  onClose: () => void;
  onSuccess: (connectionId: string) => void;
}

const providerLabels: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  ollama: 'Ollama',
  'lm-studio': 'LM Studio',
  'openai-compatible': 'OpenAI Compatible',
};

export const AddConnectionModal: React.FC<AddConnectionModalProps> = ({
  providerId: initialProviderId,
  onClose,
  onSuccess,
}) => {
  const [providerId, setProviderId] = useState(initialProviderId);
  const [name, setName] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [baseUrl, setBaseUrl] = useState(
    initialProviderId === 'ollama' ? 'http://localhost:11434' : 
    initialProviderId === 'lm-studio' ? 'http://localhost:1234' : ''
  );
  const [creating, setCreating] = useState(false);
  const { toast } = useToast();

  // Update base URL when provider changes
  const handleProviderChange = (newProviderId: string) => {
    setProviderId(newProviderId);
    if (newProviderId === 'ollama') {
      setBaseUrl('http://localhost:11434');
    } else if (newProviderId === 'lm-studio') {
      setBaseUrl('http://localhost:1234');
    } else {
      setBaseUrl('');
    }
    // Clear API key when switching providers
    setApiKey('');
  };

  const handleCreate = async () => {
    if (!name.trim()) {
      toast({
        title: 'Error',
        description: 'Please enter a connection name',
        variant: 'destructive',
      });
      return;
    }

    if (providerId !== 'ollama' && providerId !== 'lm-studio' && providerId !== 'openai-compatible' && !apiKey.trim()) {
      toast({
        title: 'Error',
        description: 'Please enter an API key',
        variant: 'destructive',
      });
      return;
    }

    setCreating(true);
    try {
      const config: any = {};
      
      if (apiKey) {
        config.api_key = apiKey;
      }
      
      if (baseUrl) {
        config.base_url = baseUrl;
      }

      // Map frontend provider IDs to backend types
      let backendProviderId = providerId;
      if (providerId === 'lm-studio') {
        backendProviderId = 'openai-compatible';
      }
      // openai-compatible is already the correct backend type

      const connection = await api.createConnection({
        provider_id: backendProviderId,
        name: name.trim(),
        config,
      });

      toast({
        title: 'Success',
        description: 'Connection created successfully',
      });

      onSuccess(connection.id);
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to create connection',
        variant: 'destructive',
      });
    } finally {
      setCreating(false);
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New Connection</DialogTitle>
          <DialogDescription>
            Create a new connection to an AI provider
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div>
            <Label htmlFor="provider">Provider</Label>
            <Select value={providerId} onValueChange={handleProviderChange}>
              <SelectTrigger id="provider">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai">
                  <div className="flex items-center gap-2">
                    <span>⚡</span>
                    <span>OpenAI</span>
                  </div>
                </SelectItem>
                <SelectItem value="anthropic">
                  <div className="flex items-center gap-2">
                    <span>🧠</span>
                    <span>Anthropic</span>
                  </div>
                </SelectItem>
                <SelectItem value="ollama">
                  <div className="flex items-center gap-2">
                    <span>🦙</span>
                    <span>Ollama (Local)</span>
                  </div>
                </SelectItem>
                <SelectItem value="lm-studio">
                  <div className="flex items-center gap-2">
                    <span>🖥️</span>
                    <span>LM Studio (Local)</span>
                  </div>
                </SelectItem>
                <SelectItem value="openai-compatible">
                  <div className="flex items-center gap-2">
                    <span>🔧</span>
                    <span>OpenAI Compatible</span>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label htmlFor="connection-name">Connection Name</Label>
            <Input
              id="connection-name"
              placeholder="e.g., Work API, Personal Project"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
            <p className="text-sm text-foreground-tertiary mt-1">
              This helps you organize multiple API keys for different purposes
            </p>
          </div>

          {providerId !== 'ollama' && providerId !== 'lm-studio' && providerId !== 'openai-compatible' && (
            <div>
              <Label htmlFor="api-key">API Key</Label>
              <Input
                id="api-key"
                type="password"
                placeholder={providerId === 'openai' ? 'sk-...' : 'Your API key'}
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
              />
              {providerId === 'openai' && (
                <p className="text-sm text-foreground-tertiary mt-1">
                  Get your API key from{' '}
                  <a
                    href="https://platform.openai.com/api-keys"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-accent-primary hover:underline"
                  >
                    platform.openai.com
                  </a>
                </p>
              )}
              {providerId === 'anthropic' && (
                <p className="text-sm text-foreground-tertiary mt-1">
                  Get your API key from{' '}
                  <a
                    href="https://console.anthropic.com/account/keys"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-accent-primary hover:underline"
                  >
                    console.anthropic.com
                  </a>
                </p>
              )}
            </div>
          )}

          {(providerId === 'ollama' || providerId === 'lm-studio' || providerId === 'openai-compatible') && (
            <div>
              <Label htmlFor="base-url">Base URL</Label>
              <Input
                id="base-url"
                placeholder="http://localhost:11434"
                value={baseUrl}
                onChange={(e) => setBaseUrl(e.target.value)}
              />
              <p className="text-sm text-foreground-tertiary mt-1">
                The URL where your {providerLabels[providerId]} server is running
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={creating}>
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={creating}>
            {creating && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
            Create Connection
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};