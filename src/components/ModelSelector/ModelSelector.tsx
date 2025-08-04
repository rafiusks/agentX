import { ChevronDown, Circle } from 'lucide-react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'

interface ModelSelectorProps {
  providers: Array<{
    id: string
    name: string
    enabled: boolean
    status: string
  }>
  currentProvider: string | null
  onProviderChange: (provider: string) => void
}

export function ModelSelector({ providers, currentProvider, onProviderChange }: ModelSelectorProps) {
  const current = providers.find(p => p.id === currentProvider)
  
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button className="btn-secondary px-4 py-2 gap-3 min-w-[180px] justify-between">
          <div className="flex items-center gap-2">
            <Circle 
              size={8} 
              className={`fill-current ${
                current?.enabled ? 'text-accent-green' : 'text-foreground-muted'
              }`}
            />
            <span className="text-sm">
              {current?.name || 'Select Model'}
            </span>
          </div>
          <ChevronDown size={14} className="text-foreground-secondary" />
        </button>
      </DropdownMenu.Trigger>

      <DropdownMenu.Portal>
        <DropdownMenu.Content
          className="min-w-[220px] bg-background-secondary rounded-md p-1 shadow-xl border border-border-subtle animate-in"
          sideOffset={5}
        >
          {providers.map(provider => (
            <DropdownMenu.Item
              key={provider.id}
              className={`
                flex items-center justify-between px-3 py-2.5 rounded-md text-sm
                outline-none cursor-pointer transition-colors
                ${provider.enabled 
                  ? 'hover:bg-background-tertiary text-foreground-primary' 
                  : 'text-foreground-muted opacity-60'
                }
                ${currentProvider === provider.id ? 'bg-background-tertiary' : ''}
              `}
              onSelect={() => provider.enabled && onProviderChange(provider.id)}
              disabled={!provider.enabled}
            >
              <div className="flex items-center gap-2">
                <Circle 
                  size={8} 
                  className={`fill-current ${
                    provider.enabled ? 'text-accent-green' : 'text-foreground-muted'
                  }`}
                />
                <span>{provider.name}</span>
              </div>
              <span className="text-xs text-foreground-muted">
                {provider.status}
              </span>
            </DropdownMenu.Item>
          ))}
          
          <DropdownMenu.Separator className="h-px bg-border-subtle my-1" />
          
          <DropdownMenu.Item
            className="px-3 py-2 text-xs text-foreground-secondary outline-none"
            onSelect={(e) => e.preventDefault()}
          >
            Tip: Add API keys in Settings
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}