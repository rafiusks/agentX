import { Zap, Grid, Code } from 'lucide-react'
import { useUIStore, UIMode } from '../../stores/ui.store'

export function ModeSelector() {
  const { mode, setMode } = useUIStore()
  
  const modes = [
    {
      id: 'simple' as UIMode,
      name: 'Simple',
      icon: Zap,
      description: 'Clean, focused chat interface',
      features: ['Basic chat', 'Provider selection', 'Essential features only']
    },
    {
      id: 'mission-control' as UIMode,
      name: 'Mission Control',
      icon: Grid,
      description: 'Advanced features and controls',
      features: ['Multiple sessions', 'Advanced settings', 'Command palette', 'Keyboard shortcuts']
    },
    {
      id: 'pro' as UIMode,
      name: 'Pro',
      icon: Code,
      description: 'Full power user interface',
      features: ['Debug info', 'Raw API responses', 'Custom prompts', 'Advanced analytics']
    }
  ]
  
  return (
    <div className="p-6">
      <h3 className="text-lg font-semibold text-foreground-primary mb-4">
        Interface Mode
      </h3>
      
      <div className="grid grid-cols-3 gap-4">
        {modes.map((modeOption) => {
          const Icon = modeOption.icon
          const isActive = mode === modeOption.id
          
          return (
            <button
              key={modeOption.id}
              onClick={() => setMode(modeOption.id)}
              className={`
                p-4 rounded-lg border transition-all
                ${isActive 
                  ? 'border-accent-blue bg-accent-blue/10' 
                  : 'border-border-subtle bg-background-tertiary hover:border-border-default'
                }
              `}
            >
              <div className="flex flex-col items-center text-center space-y-3">
                <div className={`
                  p-3 rounded-lg
                  ${isActive ? 'bg-accent-blue/20' : 'bg-background-secondary'}
                `}>
                  <Icon size={24} className={isActive ? 'text-accent-blue' : 'text-foreground-secondary'} />
                </div>
                
                <div>
                  <h4 className={`font-medium ${isActive ? 'text-accent-blue' : 'text-foreground-primary'}`}>
                    {modeOption.name}
                  </h4>
                  <p className="text-xs text-foreground-muted mt-1">
                    {modeOption.description}
                  </p>
                </div>
                
                <ul className="text-xs text-left space-y-1 w-full">
                  {modeOption.features.map((feature, i) => (
                    <li key={i} className="flex items-start gap-1">
                      <span className={`
                        inline-block w-1.5 h-1.5 rounded-full mt-1
                        ${isActive ? 'bg-accent-blue' : 'bg-foreground-muted'}
                      `} />
                      <span className="text-foreground-secondary">{feature}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </button>
          )
        })}
      </div>
      
      <div className="mt-4 p-3 rounded-lg bg-background-tertiary border border-border-subtle">
        <p className="text-sm text-foreground-secondary">
          <strong>Tip:</strong> You can quickly switch modes using the command palette (⌘K)
        </p>
      </div>
    </div>
  )
}