import { useState } from 'react'
import { ChevronRight, Sparkles, Key, Cpu, Check } from 'lucide-react'
import { invoke } from '@tauri-apps/api/core'

interface WelcomeProps {
  onComplete: () => void
}

export function Welcome({ onComplete }: WelcomeProps) {
  const [currentStep, setCurrentStep] = useState(0)
  const [apiKeys, setApiKeys] = useState({
    openai: '',
    anthropic: ''
  })
  const [saving, setSaving] = useState(false)

  const steps = [
    {
      title: 'Welcome to AgentX',
      icon: Sparkles,
      content: (
        <div className="space-y-6">
          <p className="text-lg text-foreground-secondary">
            Your AI IDE for agentic software development
          </p>
          <div className="space-y-4">
            <div className="flex items-start gap-3">
              <div className="w-2 h-2 rounded-full bg-accent-blue mt-2" />
              <div>
                <h4 className="font-medium text-foreground-primary">Multi-Provider Support</h4>
                <p className="text-sm text-foreground-muted">Use OpenAI, Anthropic, Ollama, or our Demo mode</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <div className="w-2 h-2 rounded-full bg-accent-green mt-2" />
              <div>
                <h4 className="font-medium text-foreground-primary">Lightning Fast</h4>
                <p className="text-sm text-foreground-muted">Built with Rust for &lt;50ms startup time</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <div className="w-2 h-2 rounded-full bg-accent-purple mt-2" />
              <div>
                <h4 className="font-medium text-foreground-primary">Privacy First</h4>
                <p className="text-sm text-foreground-muted">Your keys, your data, always secure</p>
              </div>
            </div>
          </div>
        </div>
      )
    },
    {
      title: 'Configure Providers',
      icon: Key,
      content: (
        <div className="space-y-6">
          <p className="text-foreground-secondary">
            Add API keys for cloud providers (optional)
          </p>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground-secondary mb-2">
                OpenAI API Key
              </label>
              <input
                type="password"
                value={apiKeys.openai}
                onChange={(e) => setApiKeys(prev => ({ ...prev, openai: e.target.value }))}
                placeholder="sk-..."
                className="w-full px-4 py-2 bg-background-tertiary rounded-lg
                         border border-border-subtle focus:border-accent-blue/50
                         focus:outline-none transition-colors
                         placeholder:text-foreground-muted"
              />
              <p className="text-xs text-foreground-muted mt-1">
                Get your key from platform.openai.com
              </p>
            </div>
            
            <div>
              <label className="block text-sm font-medium text-foreground-secondary mb-2">
                Anthropic API Key
              </label>
              <input
                type="password"
                value={apiKeys.anthropic}
                onChange={(e) => setApiKeys(prev => ({ ...prev, anthropic: e.target.value }))}
                placeholder="sk-ant-..."
                className="w-full px-4 py-2 bg-background-tertiary rounded-lg
                         border border-border-subtle focus:border-accent-blue/50
                         focus:outline-none transition-colors
                         placeholder:text-foreground-muted"
              />
              <p className="text-xs text-foreground-muted mt-1">
                Get your key from console.anthropic.com
              </p>
            </div>
          </div>
          
          <div className="p-4 bg-background-tertiary rounded-lg border border-border-subtle">
            <p className="text-sm text-foreground-secondary">
              <strong>Note:</strong> You can skip this step and use Demo mode or Ollama for local models
            </p>
          </div>
        </div>
      )
    },
    {
      title: 'Local Models',
      icon: Cpu,
      content: (
        <div className="space-y-6">
          <p className="text-foreground-secondary">
            Run models locally with Ollama (optional)
          </p>
          
          <div className="space-y-4">
            <div className="p-4 bg-background-tertiary rounded-lg border border-border-subtle">
              <h4 className="font-medium text-foreground-primary mb-2">Install Ollama</h4>
              <p className="text-sm text-foreground-secondary mb-3">
                Download from ollama.ai and run:
              </p>
              <code className="block p-3 bg-background-secondary rounded text-sm font-mono">
                ollama pull llama2
              </code>
            </div>
            
            <div className="flex items-start gap-3">
              <Check size={20} className="text-accent-green mt-0.5" />
              <div>
                <h4 className="font-medium text-foreground-primary">Privacy</h4>
                <p className="text-sm text-foreground-muted">All processing happens locally</p>
              </div>
            </div>
            
            <div className="flex items-start gap-3">
              <Check size={20} className="text-accent-green mt-0.5" />
              <div>
                <h4 className="font-medium text-foreground-primary">No API Keys</h4>
                <p className="text-sm text-foreground-muted">Works without any configuration</p>
              </div>
            </div>
            
            <div className="flex items-start gap-3">
              <Check size={20} className="text-accent-green mt-0.5" />
              <div>
                <h4 className="font-medium text-foreground-primary">Multiple Models</h4>
                <p className="text-sm text-foreground-muted">Use Llama, Mistral, CodeLlama, and more</p>
              </div>
            </div>
          </div>
        </div>
      )
    }
  ]

  const handleNext = async () => {
    if (currentStep === 1 && (apiKeys.openai || apiKeys.anthropic)) {
      setSaving(true)
      try {
        if (apiKeys.openai) {
          await invoke('update_api_key', { 
            providerId: 'openai', 
            apiKey: apiKeys.openai 
          })
        }
        if (apiKeys.anthropic) {
          await invoke('update_api_key', { 
            providerId: 'anthropic', 
            apiKey: apiKeys.anthropic 
          })
        }
      } catch (error) {
        console.error('Failed to save API keys:', error)
      }
      setSaving(false)
    }

    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1)
    } else {
      // Save completion state
      localStorage.setItem('agentx-welcome-completed', 'true')
      onComplete()
    }
  }

  const handleSkip = () => {
    localStorage.setItem('agentx-welcome-completed', 'true')
    onComplete()
  }

  const currentStepData = steps[currentStep]
  const Icon = currentStepData.icon

  return (
    <div className="min-h-screen bg-background-primary flex items-center justify-center p-8">
      <div className="max-w-2xl w-full">
        {/* Progress */}
        <div className="flex items-center gap-3 mb-8">
          {steps.map((_, index) => (
            <div
              key={index}
              className={`flex-1 h-1 rounded-full transition-colors ${
                index <= currentStep 
                  ? 'bg-accent-blue' 
                  : 'bg-background-tertiary'
              }`}
            />
          ))}
        </div>

        {/* Content */}
        <div className="glass p-8 rounded-2xl border border-border-subtle">
          <div className="flex items-center gap-3 mb-6">
            <div className="p-3 rounded-xl bg-accent-blue/10">
              <Icon size={24} className="text-accent-blue" />
            </div>
            <h2 className="text-2xl font-semibold text-foreground-primary">
              {currentStepData.title}
            </h2>
          </div>

          <div className="mb-8">
            {currentStepData.content}
          </div>

          {/* Actions */}
          <div className="flex items-center justify-between">
            <button
              onClick={handleSkip}
              className="text-sm text-foreground-muted hover:text-foreground-secondary transition-colors"
            >
              Skip setup
            </button>

            <div className="flex items-center gap-3">
              {currentStep > 0 && (
                <button
                  onClick={() => setCurrentStep(currentStep - 1)}
                  className="px-4 py-2 text-foreground-secondary hover:text-foreground-primary transition-colors"
                >
                  Back
                </button>
              )}
              
              <button
                onClick={handleNext}
                disabled={saving}
                className="px-6 py-2 bg-accent-blue text-white rounded-lg
                         hover:bg-accent-blue/90 transition-colors
                         disabled:opacity-50 disabled:cursor-not-allowed
                         flex items-center gap-2"
              >
                {saving ? (
                  <>Saving...</>
                ) : currentStep === steps.length - 1 ? (
                  <>Get Started</>
                ) : (
                  <>
                    Next
                    <ChevronRight size={16} />
                  </>
                )}
              </button>
            </div>
          </div>
        </div>

        {/* Step indicator */}
        <div className="flex items-center justify-center gap-2 mt-6">
          {steps.map((_, index) => (
            <div
              key={index}
              className={`w-2 h-2 rounded-full transition-colors ${
                index === currentStep 
                  ? 'bg-accent-blue' 
                  : 'bg-background-tertiary'
              }`}
            />
          ))}
        </div>
      </div>
    </div>
  )
}