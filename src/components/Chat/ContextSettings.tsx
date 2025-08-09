import { memo, useState } from 'react';
import { Settings, Brain, Clock, Layers } from 'lucide-react';
import { DEFAULT_CONTEXT_CONFIG } from '../../config/context';

interface ContextSettingsProps {
  onStrategyChange: (strategy: 'sliding_window' | 'smart' | 'full') => void;
  currentStrategy: string;
}

export const ContextSettings = memo(function ContextSettings({
  onStrategyChange,
  currentStrategy
}: ContextSettingsProps) {
  const [isOpen, setIsOpen] = useState(false);

  const strategies = [
    {
      id: 'sliding_window',
      name: 'Recent',
      icon: Clock,
      description: 'Keep most recent messages only',
      color: 'text-blue-500',
    },
    {
      id: 'smart',
      name: 'Smart',
      icon: Brain,
      description: 'Prioritize important messages',
      color: 'text-green-500',
    },
    {
      id: 'full',
      name: 'Full',
      icon: Layers,
      description: 'Include all messages (may hit limits)',
      color: 'text-amber-500',
    },
  ];

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="p-2 rounded-lg hover:bg-background-secondary transition-colors"
        title="Context Settings"
      >
        <Settings size={16} className="text-foreground-muted" />
      </button>

      {isOpen && (
        <>
          <div 
            className="fixed inset-0 z-40" 
            onClick={() => setIsOpen(false)}
          />
          <div className="absolute bottom-full right-0 mb-2 z-50 
                        bg-background-primary border border-border-subtle 
                        rounded-lg shadow-lg p-3 min-w-[250px]">
            <div className="text-sm font-medium text-foreground-primary mb-2">
              Context Strategy
            </div>
            
            <div className="space-y-1">
              {strategies.map((strategy) => {
                const Icon = strategy.icon;
                const isActive = currentStrategy === strategy.id;
                
                return (
                  <button
                    key={strategy.id}
                    onClick={() => {
                      onStrategyChange(strategy.id as any);
                      setIsOpen(false);
                    }}
                    className={`
                      w-full flex items-start gap-3 p-2 rounded-lg
                      transition-all duration-200
                      ${isActive 
                        ? 'bg-accent-blue/10 border border-accent-blue/20' 
                        : 'hover:bg-background-secondary'
                      }
                    `}
                  >
                    <Icon size={16} className={strategy.color} />
                    <div className="flex-1 text-left">
                      <div className="text-sm font-medium text-foreground-primary">
                        {strategy.name}
                      </div>
                      <div className="text-xs text-foreground-muted">
                        {strategy.description}
                      </div>
                    </div>
                    {isActive && (
                      <div className="w-2 h-2 rounded-full bg-accent-blue mt-1" />
                    )}
                  </button>
                );
              })}
            </div>

            <div className="mt-3 pt-3 border-t border-border-subtle">
              <div className="text-xs text-foreground-muted">
                Smart mode uses AI to identify important messages with code, 
                errors, or key decisions to keep in context.
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
});