import { useState, useEffect } from 'react'
import { invoke } from '@tauri-apps/api/core'
import { Key, Save, Eye, EyeOff, Server, Cpu, RefreshCw } from 'lucide-react'
import { ModeSelector } from '../ModeSelector/ModeSelector'

interface SettingsProps {
  providers: Array<{
    id: string
    name: string
    enabled: boolean
    status: string
  }>
  onProvidersUpdate: () => void
}

interface ProviderSettings {
  base_url: string
  models: Array<{
    name: string
    display_name: string
    context_size: number
    supports_streaming: boolean
    description?: string
  }>
  default_model?: string
}

interface DiscoveredModel {
  id: string
  name: string
  display_name: string
  context_size: number
  supports_streaming: boolean
  description: string
}

export function Settings({ providers, onProvidersUpdate }: SettingsProps) {
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({})
  const [showKeys, setShowKeys] = useState<Record<string, boolean>>({})
  const [saving, setSaving] = useState<string | null>(null)
  const [providerSettings, setProviderSettings] = useState<Record<string, ProviderSettings>>({})
  const [editedSettings, setEditedSettings] = useState<Record<string, { base_url?: string; default_model?: string }>>({})
  const [discoveredModels, setDiscoveredModels] = useState<Record<string, DiscoveredModel[]>>({})
  const [discoveringModels, setDiscoveringModels] = useState<Record<string, boolean>>({})
  const [discoveryError, setDiscoveryError] = useState<Record<string, string>>({})

  useEffect(() => {
    // Load settings for each provider
    providers.forEach(async (provider) => {
      if (provider.id === 'local' || provider.id === 'ollama') {
        try {
          const settings = await invoke<ProviderSettings>('get_provider_settings', {
            providerId: provider.id
          })
          setProviderSettings(prev => ({ ...prev, [provider.id]: settings }))
          
          // Automatically discover models for OpenAI-compatible providers
          discoverModels(provider.id, settings.base_url)
        } catch (error) {
          console.error(`Failed to load settings for ${provider.id}:`, error)
        }
      }
    })
  }, [providers])

  const discoverModels = async (providerId: string, baseUrl?: string) => {
    setDiscoveringModels(prev => ({ ...prev, [providerId]: true }))
    setDiscoveryError(prev => ({ ...prev, [providerId]: '' }))
    
    try {
      const models = await invoke<DiscoveredModel[]>('discover_models', {
        providerId,
        baseUrl: baseUrl || editedSettings[providerId]?.base_url
      })
      
      setDiscoveredModels(prev => ({ ...prev, [providerId]: models }))
    } catch (error) {
      console.error(`Failed to discover models for ${providerId}:`, error)
      setDiscoveryError(prev => ({ ...prev, [providerId]: String(error) }))
    } finally {
      setDiscoveringModels(prev => ({ ...prev, [providerId]: false }))
    }
  }

  const handleSaveKey = async (providerId: string) => {
    const apiKey = apiKeys[providerId]
    if (!apiKey) return
    
    setSaving(providerId)
    
    try {
      await invoke('update_api_key', {
        providerId,
        apiKey
      })
      
      // Clear the key from state after saving
      setApiKeys(prev => ({ ...prev, [providerId]: '' }))
      
      // Refresh providers
      await onProvidersUpdate()
    } catch (error) {
      console.error('Failed to save API key:', error)
    } finally {
      setSaving(null)
    }
  }

  const handleSaveSettings = async (providerId: string) => {
    const settings = editedSettings[providerId]
    if (!settings || (!settings.base_url && !settings.default_model)) return
    
    setSaving(providerId)
    
    try {
      await invoke('update_provider_settings', {
        providerId,
        baseUrl: settings.base_url || undefined,
        defaultModel: settings.default_model || undefined
      })
      
      // Clear edited settings
      setEditedSettings(prev => ({ ...prev, [providerId]: {} }))
      
      // Refresh providers and settings
      await onProvidersUpdate()
      const newSettings = await invoke<ProviderSettings>('get_provider_settings', {
        providerId
      })
      setProviderSettings(prev => ({ ...prev, [providerId]: newSettings }))
      
      // Re-discover models if base URL changed
      if (settings.base_url) {
        discoverModels(providerId, settings.base_url)
      }
    } catch (error) {
      console.error('Failed to save settings:', error)
    } finally {
      setSaving(null)
    }
  }

  const providerInfo = {
    openai: {
      description: 'Access GPT-4, GPT-3.5, and other OpenAI models',
      helpUrl: 'https://platform.openai.com/api-keys',
      placeholder: 'sk-...'
    },
    anthropic: {
      description: 'Access Claude 3 Opus, Sonnet, and Haiku models',
      helpUrl: 'https://console.anthropic.com/settings/keys',
      placeholder: 'sk-ant-...'
    },
    local: {
      description: 'Run models locally with Ollama, LM Studio, or llama.cpp',
      helpUrl: 'https://ollama.ai',
      placeholder: 'No API key needed',
      configurable: true
    },
    ollama: {
      description: 'Run models locally with Ollama',
      helpUrl: 'https://ollama.ai',
      placeholder: 'No API key needed',
      configurable: true
    },
    demo: {
      description: 'Built-in demo provider for testing',
      helpUrl: null,
      placeholder: 'Always available'
    }
  }

  return (
    <div className="max-w-4xl mx-auto p-8 space-y-8">
      {/* Interface Mode */}
      <div>
        <h2 className="text-2xl font-semibold mb-2">Interface</h2>
        <p className="text-foreground-muted mb-6">
          Choose your preferred interface complexity
        </p>
        <ModeSelector />
      </div>
      
      {/* Providers */}
      <div>
        <h2 className="text-2xl font-semibold mb-2">AI Providers</h2>
        <p className="text-foreground-muted mb-6">
          Configure your AI providers and API keys
        </p>
        
        <div className="space-y-6">
        {providers.map(provider => {
          const info = providerInfo[provider.id as keyof typeof providerInfo]
          const needsApiKey = provider.id !== 'local' && provider.id !== 'ollama' && provider.id !== 'demo'
          const settings = providerSettings[provider.id]
          const edited = editedSettings[provider.id] || {}
          const discovered = discoveredModels[provider.id] || []
          const isDiscovering = discoveringModels[provider.id] || false
          const error = discoveryError[provider.id]
          
          // Use discovered models if available, otherwise fall back to configured models
          const availableModels = discovered.length > 0 ? discovered : (settings?.models || [])
          
          return (
            <div key={provider.id} className="card">
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h3 className="text-lg font-medium flex items-center gap-2">
                    {provider.name}
                    <span className={`text-xs px-2 py-0.5 rounded-full ${
                      provider.enabled 
                        ? 'bg-accent-green/20 text-accent-green' 
                        : 'bg-foreground-muted/20 text-foreground-muted'
                    }`}>
                      {provider.status}
                    </span>
                  </h3>
                  <p className="text-sm text-foreground-muted mt-1">
                    {info?.description}
                  </p>
                </div>
                
                {info?.helpUrl && (
                  <a
                    href={info.helpUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm text-accent-blue hover:underline"
                  >
                    Get Started →
                  </a>
                )}
              </div>
              
              {/* API Key Input */}
              {needsApiKey && (
                <div className="flex items-center gap-2 mb-4">
                  <div className="relative flex-1">
                    <Key size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted" />
                    <input
                      type={showKeys[provider.id] ? 'text' : 'password'}
                      value={apiKeys[provider.id] || ''}
                      onChange={(e) => setApiKeys(prev => ({ 
                        ...prev, 
                        [provider.id]: e.target.value 
                      }))}
                      placeholder={info?.placeholder || 'Enter API key'}
                      className="input pl-10 pr-10"
                    />
                    <button
                      onClick={() => setShowKeys(prev => ({ 
                        ...prev, 
                        [provider.id]: !prev[provider.id] 
                      }))}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-foreground-muted
                               hover:text-foreground-primary transition-colors"
                    >
                      {showKeys[provider.id] ? <EyeOff size={16} /> : <Eye size={16} />}
                    </button>
                  </div>
                  
                  <button
                    onClick={() => handleSaveKey(provider.id)}
                    disabled={!apiKeys[provider.id] || saving === provider.id}
                    className="btn-primary px-4 py-2 gap-2"
                  >
                    <Save size={16} />
                    {saving === provider.id ? 'Saving...' : 'Save'}
                  </button>
                </div>
              )}
              
              {/* Configuration for Local Providers */}
              {info?.configurable && settings && (
                <div className="space-y-4 mt-4 pt-4 border-t border-border-subtle">
                  {/* Base URL */}
                  <div>
                    <label className="text-sm font-medium mb-1 block">
                      API Endpoint
                    </label>
                    <div className="relative">
                      <Server size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted" />
                      <input
                        type="text"
                        value={edited.base_url ?? settings.base_url}
                        onChange={(e) => setEditedSettings(prev => ({ 
                          ...prev, 
                          [provider.id]: { ...prev[provider.id], base_url: e.target.value }
                        }))}
                        placeholder="http://localhost:11434"
                        className="input pl-10 w-full"
                      />
                    </div>
                    <p className="text-xs text-foreground-muted mt-1">
                      Default ports: Ollama (11434), LM Studio (1234), llama.cpp (8080)
                    </p>
                  </div>
                  
                  {/* Model Selection */}
                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <label className="text-sm font-medium">
                        Default Model
                      </label>
                      <button
                        onClick={() => discoverModels(provider.id, edited.base_url || settings.base_url)}
                        disabled={isDiscovering}
                        className="text-xs text-accent-blue hover:underline flex items-center gap-1"
                      >
                        <RefreshCw size={12} className={isDiscovering ? 'animate-spin' : ''} />
                        {isDiscovering ? 'Discovering...' : 'Refresh Models'}
                      </button>
                    </div>
                    <div className="relative">
                      <Cpu size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted" />
                      <select
                        value={edited.default_model ?? settings.default_model ?? ''}
                        onChange={(e) => setEditedSettings(prev => ({ 
                          ...prev, 
                          [provider.id]: { ...prev[provider.id], default_model: e.target.value }
                        }))}
                        className="input pl-10 w-full"
                      >
                        <option value="">Select a model</option>
                        {availableModels.map(model => (
                          <option key={model.name} value={model.name}>
                            {model.display_name || model.name}
                          </option>
                        ))}
                      </select>
                    </div>
                    {error && (
                      <p className="text-xs text-red-500 mt-1">
                        Failed to discover models: {error}
                      </p>
                    )}
                    {discovered.length > 0 && (
                      <p className="text-xs text-accent-green mt-1">
                        Discovered {discovered.length} models from the API
                      </p>
                    )}
                  </div>
                  
                  {/* Save Settings Button */}
                  {(edited.base_url || edited.default_model) && (
                    <button
                      onClick={() => handleSaveSettings(provider.id)}
                      disabled={saving === provider.id}
                      className="btn-primary px-4 py-2 gap-2"
                    >
                      <Save size={16} />
                      {saving === provider.id ? 'Saving...' : 'Save Settings'}
                    </button>
                  )}
                </div>
              )}
            </div>
          )
        })}
        </div>
        
        <div className="mt-12 p-6 bg-background-tertiary rounded-lg border border-border-subtle">
          <h3 className="text-sm font-medium mb-2">Privacy & Security</h3>
          <p className="text-sm text-foreground-muted">
            • API keys are stored securely in your system's credential manager<br/>
            • Keys are never sent to our servers - AgentX is fully local<br/>
            • You can revoke keys anytime from your provider's dashboard<br/>
            • Local models are discovered dynamically from the API endpoint
          </p>
        </div>
      </div>
    </div>
  )
}