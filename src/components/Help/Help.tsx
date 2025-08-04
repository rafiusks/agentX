import { X, Command, Search, MessageSquare, Settings, Info } from 'lucide-react'

interface HelpProps {
  open: boolean
  onClose: () => void
}

export function Help({ open, onClose }: HelpProps) {
  if (!open) return null

  const shortcuts = [
    { keys: ['⌘', 'K'], description: 'Open command palette' },
    { keys: ['⌘', 'N'], description: 'New chat session' },
    { keys: ['⌘', ','], description: 'Open settings' },
    { keys: ['⌘', '1'], description: 'Switch to Chat tab' },
    { keys: ['⌘', '2'], description: 'Switch to Agents tab' },
    { keys: ['⌘', '3'], description: 'Switch to Settings tab' },
    { keys: ['F1'], description: 'Show this help' },
    { keys: ['Esc'], description: 'Close dialogs' },
    { keys: ['Enter'], description: 'Send message (in chat)' },
    { keys: ['Shift', 'Enter'], description: 'New line (in chat)' },
  ]

  const features = [
    {
      icon: MessageSquare,
      title: 'Multi-Provider Chat',
      description: 'Chat with OpenAI, Anthropic, Ollama, or Demo mode'
    },
    {
      icon: Command,
      title: 'Command Palette',
      description: 'Quick access to all features with ⌘K'
    },
    {
      icon: Search,
      title: 'Smart Search',
      description: 'Search through conversations and commands'
    },
    {
      icon: Settings,
      title: 'Customizable',
      description: 'Configure API keys and preferences'
    }
  ]

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-8">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />
      
      {/* Content */}
      <div className="relative max-w-4xl w-full max-h-[80vh] overflow-hidden rounded-2xl bg-background-secondary border border-border-subtle shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-border-subtle">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-accent-blue/10">
              <Info size={20} className="text-accent-blue" />
            </div>
            <h2 className="text-xl font-semibold text-foreground-primary">
              AgentX Help
            </h2>
          </div>
          <button
            onClick={onClose}
            className="p-2 rounded-lg hover:bg-background-tertiary transition-colors"
          >
            <X size={20} className="text-foreground-secondary" />
          </button>
        </div>
        
        {/* Body */}
        <div className="overflow-y-auto p-6 space-y-8" style={{ maxHeight: 'calc(80vh - 80px)' }}>
          {/* Keyboard Shortcuts */}
          <section>
            <h3 className="text-lg font-semibold text-foreground-primary mb-4">
              Keyboard Shortcuts
            </h3>
            <div className="grid grid-cols-2 gap-3">
              {shortcuts.map((shortcut, index) => (
                <div 
                  key={index}
                  className="flex items-center justify-between p-3 rounded-lg bg-background-tertiary"
                >
                  <span className="text-sm text-foreground-secondary">
                    {shortcut.description}
                  </span>
                  <div className="flex items-center gap-1">
                    {shortcut.keys.map((key, i) => (
                      <span key={i} className="flex items-center gap-1">
                        <kbd className="px-2 py-1 text-xs font-medium bg-background-primary 
                                     rounded border border-border-subtle">
                          {key}
                        </kbd>
                        {i < shortcut.keys.length - 1 && (
                          <span className="text-foreground-muted">+</span>
                        )}
                      </span>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </section>
          
          {/* Features */}
          <section>
            <h3 className="text-lg font-semibold text-foreground-primary mb-4">
              Features
            </h3>
            <div className="grid grid-cols-2 gap-4">
              {features.map((feature, index) => {
                const Icon = feature.icon
                return (
                  <div 
                    key={index}
                    className="p-4 rounded-lg bg-background-tertiary border border-border-subtle"
                  >
                    <div className="flex items-start gap-3">
                      <div className="p-2 rounded-lg bg-background-primary">
                        <Icon size={16} className="text-accent-blue" />
                      </div>
                      <div>
                        <h4 className="font-medium text-foreground-primary mb-1">
                          {feature.title}
                        </h4>
                        <p className="text-sm text-foreground-secondary">
                          {feature.description}
                        </p>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          </section>
          
          {/* Tips */}
          <section>
            <h3 className="text-lg font-semibold text-foreground-primary mb-4">
              Pro Tips
            </h3>
            <div className="space-y-3">
              <div className="p-3 rounded-lg bg-accent-blue/10 border border-accent-blue/20">
                <p className="text-sm text-foreground-secondary">
                  <strong className="text-accent-blue">Tip:</strong> Use the Demo provider to explore AgentX without any API keys
                </p>
              </div>
              <div className="p-3 rounded-lg bg-accent-green/10 border border-accent-green/20">
                <p className="text-sm text-foreground-secondary">
                  <strong className="text-accent-green">Tip:</strong> Install Ollama for completely private, local AI models
                </p>
              </div>
              <div className="p-3 rounded-lg bg-accent-purple/10 border border-accent-purple/20">
                <p className="text-sm text-foreground-secondary">
                  <strong className="text-accent-purple">Tip:</strong> Use ⌘K to quickly switch between providers
                </p>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}