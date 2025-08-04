import { useState } from 'react'
import { invoke } from '@tauri-apps/api/core'
import { Key, Save, Eye, EyeOff } from 'lucide-react'
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

export function Settings({ providers, onProvidersUpdate }: SettingsProps) {
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({})
  const [showKeys, setShowKeys] = useState<Record<string, boolean>>({})
  const [saving, setSaving] = useState<string | null>(null)

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
    ollama: {
      description: 'Run models locally with Ollama',
      helpUrl: 'https://ollama.ai',
      placeholder: 'No API key needed'
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
          const needsApiKey = provider.id !== 'ollama' && provider.id !== 'demo'
          
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
                    Get API Key →
                  </a>
                )}
              </div>
              
              {needsApiKey && (
                <div className="flex items-center gap-2">
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
            </div>
          )
        })}
        </div>
        
        <div className="mt-12 p-6 bg-background-tertiary rounded-lg border border-border-subtle">
          <h3 className="text-sm font-medium mb-2">Privacy & Security</h3>
          <p className="text-sm text-foreground-muted">
            • API keys are stored securely in your system's credential manager<br/>
            • Keys are never sent to our servers - AgentX is fully local<br/>
            • You can revoke keys anytime from your provider's dashboard
          </p>
        </div>
      </div>
    </div>
  )
}