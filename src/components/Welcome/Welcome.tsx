import { useState } from 'react'
import { ChevronRight, Sparkles, Key, Cpu } from 'lucide-react'
import { api } from '../../services/api'

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
                <p className="text-sm text-foreground-muted">Built with Go for fast performance</p>
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
            Add API keys for the providers you'd like to use. You can always add more later.
          </p>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground-primary mb-2">
                OpenAI API Key
              </label>
              <input
                type="password"
                value={apiKeys.openai}
                onChange={(e) => setApiKeys({ ...apiKeys, openai: e.target.value })}
                placeholder="sk-..."
                className="w-full px-4 py-2 bg-background-secondary rounded-lg border border-border-subtle focus:border-accent-primary focus:outline-none"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground-primary mb-2">
                Anthropic API Key
              </label>
              <input
                type="password"
                value={apiKeys.anthropic}
                onChange={(e) => setApiKeys({ ...apiKeys, anthropic: e.target.value })}
                placeholder="sk-ant-..."
                className="w-full px-4 py-2 bg-background-secondary rounded-lg border border-border-subtle focus:border-accent-primary focus:outline-none"
              />
            </div>
            <p className="text-sm text-foreground-muted">
              Don't have API keys? You can use Demo mode or Ollama for local models.
            </p>
          </div>
        </div>
      )
    },
    {
      title: 'Choose Your Mode',
      icon: Cpu,
      content: (
        <div className="space-y-6">
          <p className="text-foreground-secondary">
            AgentX adapts to your workflow with three progressive UI modes:
          </p>
          <div className="space-y-3">
            <button
              onClick={() => {
                localStorage.setItem('agentx-ui-mode', 'simple')
                handleComplete()
              }}
              className="w-full p-4 bg-background-secondary rounded-lg border border-border-subtle hover:border-accent-primary transition-colors text-left"
            >
              <h4 className="font-medium text-foreground-primary">Simple Mode</h4>
              <p className="text-sm text-foreground-muted mt-1">Clean chat interface, perfect for focused work</p>
            </button>
            <button
              onClick={() => {
                localStorage.setItem('agentx-ui-mode', 'terminal')
                handleComplete()
              }}
              className="w-full p-4 bg-background-secondary rounded-lg border border-border-subtle hover:border-accent-primary transition-colors text-left"
            >
              <h4 className="font-medium text-foreground-primary">Terminal Mode</h4>
              <p className="text-sm text-foreground-muted mt-1">Enhanced with command blocks and history</p>
            </button>
            <button
              onClick={() => {
                localStorage.setItem('agentx-ui-mode', 'pro')
                handleComplete()
              }}
              className="w-full p-4 bg-background-secondary rounded-lg border border-border-subtle hover:border-accent-primary transition-colors text-left"
            >
              <h4 className="font-medium text-foreground-primary">Pro Mode</h4>
              <p className="text-sm text-foreground-muted mt-1">Full IDE experience with agents and canvas</p>
            </button>
          </div>
        </div>
      )
    }
  ]

  const handleNext = async () => {
    if (currentStep === 1 && (apiKeys.openai || apiKeys.anthropic)) {
      setSaving(true)
      try {
        // Save API keys
        if (apiKeys.openai) {
          await api.updateProviderConfig('openai', { api_key: apiKeys.openai })
        }
        if (apiKeys.anthropic) {
          await api.updateProviderConfig('anthropic', { api_key: apiKeys.anthropic })
        }
      } catch (error) {
        console.error('Failed to save API keys:', error)
      } finally {
        setSaving(false)
      }
    }
    
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1)
    }
  }

  const handleComplete = () => {
    localStorage.setItem('agentx-welcome-completed', 'true')
    onComplete()
  }

  const currentStepData = steps[currentStep]
  const Icon = currentStepData.icon

  return (
    <div className="fixed inset-0 bg-background-primary flex items-center justify-center p-8">
      <div className="max-w-2xl w-full">
        {/* Progress */}
        <div className="flex gap-2 mb-8">
          {steps.map((_, index) => (
            <div
              key={index}
              className={`h-1 flex-1 rounded-full transition-colors ${
                index <= currentStep ? 'bg-accent-primary' : 'bg-border-subtle'
              }`}
            />
          ))}
        </div>

        {/* Content */}
        <div className="bg-background-secondary rounded-xl p-8 shadow-xl border border-border-subtle">
          <div className="flex items-center gap-3 mb-6">
            <div className="p-3 rounded-xl bg-accent-primary/10">
              <Icon className="w-6 h-6 text-accent-primary" />
            </div>
            <h2 className="text-2xl font-semibold text-foreground-primary">
              {currentStepData.title}
            </h2>
          </div>

          {currentStepData.content}

          {/* Actions */}
          <div className="flex justify-between mt-8">
            <button
              onClick={() => setCurrentStep(Math.max(0, currentStep - 1))}
              disabled={currentStep === 0}
              className="px-4 py-2 text-foreground-secondary hover:text-foreground-primary disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Back
            </button>

            {currentStep < steps.length - 1 ? (
              <button
                onClick={handleNext}
                disabled={saving}
                className="flex items-center gap-2 px-6 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {saving ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    Saving...
                  </>
                ) : (
                  <>
                    Next
                    <ChevronRight className="w-4 h-4" />
                  </>
                )}
              </button>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  )
}