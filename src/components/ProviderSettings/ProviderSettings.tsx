import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useToast } from '@/components/ui/use-toast';
import { Loader2, Save, RefreshCw, Check, X } from 'lucide-react';
import { api } from '@/services/api';

interface Provider {
  id: string;
  type: string;
  name: string;
  base_url?: string;
  api_key?: string;
  models: string[];
  default_model?: string;
}

interface ProviderSettingsProps {
  onClose?: () => void;
}

export const ProviderSettings: React.FC<ProviderSettingsProps> = ({ onClose }) => {
  const defaultProviders: Record<string, Provider> = {
    openai: {
      id: 'openai',
      type: 'openai',
      name: 'OpenAI',
      models: [],
    },
    anthropic: {
      id: 'anthropic',
      type: 'anthropic',
      name: 'Anthropic',
      models: [],
    },
    ollama: {
      id: 'ollama',
      type: 'ollama',
      name: 'Ollama',
      base_url: 'http://localhost:11434',
      models: [],
    },
    'lm-studio': {
      id: 'lm-studio',
      type: 'openai-compatible',
      name: 'LM Studio',
      base_url: 'http://localhost:1234',
      models: [],
    },
  };

  const [providers, setProviders] = useState<Record<string, Provider>>(defaultProviders);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<Record<string, boolean>>({});
  const [discovering, setDiscovering] = useState<Record<string, boolean>>({});
  const [testingConnection, setTestingConnection] = useState<Record<string, boolean>>({});
  const [connectionStatus, setConnectionStatus] = useState<Record<string, 'success' | 'error' | null>>({});
  const { toast } = useToast();

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const data = await api.getSettings();
      // Merge loaded settings with defaults
      const mergedProviders = { ...defaultProviders };
      if (data.providers) {
        Object.keys(data.providers).forEach(key => {
          mergedProviders[key] = {
            ...defaultProviders[key],
            ...data.providers[key],
          };
        });
      }
      setProviders(mergedProviders);
    } catch (error) {
      // Keep default providers even if loading fails
      console.error('Failed to load settings:', error);
    } finally {
      setLoading(false);
    }
  };

  const saveProvider = async (providerId: string) => {
    setSaving({ ...saving, [providerId]: true });
    
    try {
      const provider = providers[providerId];
      await api.updateProviderConfig(providerId, provider);

      toast({
        title: 'Success',
        description: `${provider.name} settings saved successfully`,
      });

      // Test connection after saving
      await testConnection(providerId);
    } catch (error) {
      toast({
        title: 'Error',
        description: `Failed to save ${providers[providerId].name} settings`,
        variant: 'destructive',
      });
    } finally {
      setSaving({ ...saving, [providerId]: false });
    }
  };

  const discoverModels = async (providerId: string) => {
    setDiscovering({ ...discovering, [providerId]: true });
    
    try {
      const models = await api.discoverModels(providerId);
      const modelIds = models.map((m: any) => m.id);
      
      setProviders({
        ...providers,
        [providerId]: {
          ...providers[providerId],
          models: modelIds,
          default_model: modelIds[0] || '',
        },
      });

      toast({
        title: 'Success',
        description: `Found ${models.length} models`,
      });
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to discover models',
        variant: 'destructive',
      });
    } finally {
      setDiscovering({ ...discovering, [providerId]: false });
    }
  };

  const testConnection = async (providerId: string) => {
    setTestingConnection({ ...testingConnection, [providerId]: true });
    setConnectionStatus({ ...connectionStatus, [providerId]: null });

    try {
      await api.discoverModels(providerId);
      setConnectionStatus({ ...connectionStatus, [providerId]: 'success' });
    } catch (error) {
      setConnectionStatus({ ...connectionStatus, [providerId]: 'error' });
    } finally {
      setTestingConnection({ ...testingConnection, [providerId]: false });
    }
  };

  const updateProvider = (providerId: string, updates: Partial<Provider>) => {
    setProviders({
      ...providers,
      [providerId]: {
        ...providers[providerId],
        ...updates,
      },
    });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="w-8 h-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="w-full max-w-4xl mx-auto p-4">
      <div className="mb-6">
        <h2 className="text-2xl font-bold">Provider Settings</h2>
        <p className="text-muted-foreground">Configure your AI provider API keys and settings</p>
      </div>

      <Tabs defaultValue="openai" className="w-full">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="openai">OpenAI</TabsTrigger>
          <TabsTrigger value="anthropic">Anthropic</TabsTrigger>
          <TabsTrigger value="ollama">Ollama</TabsTrigger>
          <TabsTrigger value="lm-studio">LM Studio</TabsTrigger>
        </TabsList>

        {/* OpenAI Settings */}
        <TabsContent value="openai">
          <Card>
            <CardHeader>
              <CardTitle>OpenAI Configuration</CardTitle>
              <CardDescription>
                Configure your OpenAI API key to use GPT-4 and GPT-3.5 models
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="openai-key">API Key</Label>
                <Input
                  id="openai-key"
                  type="password"
                  placeholder="sk-..."
                  value={providers.openai?.api_key || ''}
                  onChange={(e) => updateProvider('openai', { api_key: e.target.value })}
                />
                <p className="text-sm text-muted-foreground">
                  Get your API key from{' '}
                  <a href="https://platform.openai.com/api-keys" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                    platform.openai.com
                  </a>
                </p>
              </div>

              <div className="flex items-center gap-2">
                <Button
                  onClick={() => saveProvider('openai')}
                  disabled={saving.openai || !providers.openai?.api_key}
                >
                  {saving.openai ? (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  ) : (
                    <Save className="w-4 h-4 mr-2" />
                  )}
                  Save Settings
                </Button>

                {connectionStatus.openai && (
                  <div className="flex items-center gap-1">
                    {connectionStatus.openai === 'success' ? (
                      <>
                        <Check className="w-4 h-4 text-green-500" />
                        <span className="text-sm text-green-500">Connected</span>
                      </>
                    ) : (
                      <>
                        <X className="w-4 h-4 text-red-500" />
                        <span className="text-sm text-red-500">Connection failed</span>
                      </>
                    )}
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Anthropic Settings */}
        <TabsContent value="anthropic">
          <Card>
            <CardHeader>
              <CardTitle>Anthropic Configuration</CardTitle>
              <CardDescription>
                Configure your Anthropic API key to use Claude models
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="anthropic-key">API Key</Label>
                <Input
                  id="anthropic-key"
                  type="password"
                  placeholder="sk-ant-..."
                  value={providers.anthropic?.api_key || ''}
                  onChange={(e) => updateProvider('anthropic', { api_key: e.target.value })}
                />
                <p className="text-sm text-muted-foreground">
                  Get your API key from{' '}
                  <a href="https://console.anthropic.com/" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                    console.anthropic.com
                  </a>
                </p>
              </div>

              <div className="flex items-center gap-2">
                <Button
                  onClick={() => saveProvider('anthropic')}
                  disabled={saving.anthropic || !providers.anthropic?.api_key}
                >
                  {saving.anthropic ? (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  ) : (
                    <Save className="w-4 h-4 mr-2" />
                  )}
                  Save Settings
                </Button>

                {connectionStatus.anthropic && (
                  <div className="flex items-center gap-1">
                    {connectionStatus.anthropic === 'success' ? (
                      <>
                        <Check className="w-4 h-4 text-green-500" />
                        <span className="text-sm text-green-500">Connected</span>
                      </>
                    ) : (
                      <>
                        <X className="w-4 h-4 text-red-500" />
                        <span className="text-sm text-red-500">Connection failed</span>
                      </>
                    )}
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Ollama Settings */}
        <TabsContent value="ollama">
          <Card>
            <CardHeader>
              <CardTitle>Ollama Configuration</CardTitle>
              <CardDescription>
                Configure Ollama for local model execution
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="ollama-url">Base URL</Label>
                <Input
                  id="ollama-url"
                  type="url"
                  placeholder="http://localhost:11434"
                  value={providers.ollama?.base_url || 'http://localhost:11434'}
                  onChange={(e) => updateProvider('ollama', { base_url: e.target.value })}
                />
                <p className="text-sm text-muted-foreground">
                  Default Ollama API endpoint. Make sure Ollama is running locally.
                </p>
              </div>

              <div className="space-y-2">
                <Label>Available Models</Label>
                {providers.ollama?.models && providers.ollama.models.length > 0 ? (
                  <div className="text-sm text-muted-foreground">
                    {providers.ollama.models.join(', ')}
                  </div>
                ) : (
                  <div className="text-sm text-muted-foreground">
                    No models found. Click "Discover Models" to find available models.
                  </div>
                )}
              </div>

              <div className="flex items-center gap-2">
                <Button
                  onClick={() => saveProvider('ollama')}
                  disabled={saving.ollama}
                >
                  {saving.ollama ? (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  ) : (
                    <Save className="w-4 h-4 mr-2" />
                  )}
                  Save Settings
                </Button>

                <Button
                  variant="outline"
                  onClick={() => discoverModels('ollama')}
                  disabled={discovering.ollama}
                >
                  {discovering.ollama ? (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  ) : (
                    <RefreshCw className="w-4 h-4 mr-2" />
                  )}
                  Discover Models
                </Button>

                {connectionStatus.ollama && (
                  <div className="flex items-center gap-1">
                    {connectionStatus.ollama === 'success' ? (
                      <>
                        <Check className="w-4 h-4 text-green-500" />
                        <span className="text-sm text-green-500">Connected</span>
                      </>
                    ) : (
                      <>
                        <X className="w-4 h-4 text-red-500" />
                        <span className="text-sm text-red-500">Not available</span>
                      </>
                    )}
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* LM Studio Settings */}
        <TabsContent value="lm-studio">
          <Card>
            <CardHeader>
              <CardTitle>LM Studio Configuration</CardTitle>
              <CardDescription>
                Configure LM Studio for local model execution
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="lmstudio-url">Base URL</Label>
                <Input
                  id="lmstudio-url"
                  type="url"
                  placeholder="http://localhost:1234"
                  value={providers['lm-studio']?.base_url || 'http://localhost:1234'}
                  onChange={(e) => updateProvider('lm-studio', { base_url: e.target.value })}
                />
                <p className="text-sm text-muted-foreground">
                  Default LM Studio API endpoint. Make sure LM Studio server is running.
                </p>
              </div>

              <div className="space-y-2">
                <Label>Available Models</Label>
                {providers['lm-studio']?.models && providers['lm-studio'].models.length > 0 ? (
                  <div className="text-sm text-muted-foreground">
                    {providers['lm-studio'].models.join(', ')}
                  </div>
                ) : (
                  <div className="text-sm text-muted-foreground">
                    No models found. Click "Discover Models" to find available models.
                  </div>
                )}
              </div>

              <div className="flex items-center gap-2">
                <Button
                  onClick={() => saveProvider('lm-studio')}
                  disabled={saving['lm-studio']}
                >
                  {saving['lm-studio'] ? (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  ) : (
                    <Save className="w-4 h-4 mr-2" />
                  )}
                  Save Settings
                </Button>

                <Button
                  variant="outline"
                  onClick={() => discoverModels('lm-studio')}
                  disabled={discovering['lm-studio']}
                >
                  {discovering['lm-studio'] ? (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  ) : (
                    <RefreshCw className="w-4 h-4 mr-2" />
                  )}
                  Discover Models
                </Button>

                {connectionStatus['lm-studio'] && (
                  <div className="flex items-center gap-1">
                    {connectionStatus['lm-studio'] === 'success' ? (
                      <>
                        <Check className="w-4 h-4 text-green-500" />
                        <span className="text-sm text-green-500">Connected</span>
                      </>
                    ) : (
                      <>
                        <X className="w-4 h-4 text-red-500" />
                        <span className="text-sm text-red-500">Not available</span>
                      </>
                    )}
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {onClose && (
        <div className="mt-6 flex justify-end">
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
        </div>
      )}
    </div>
  );
};