import { MessageSquare, Boxes, Server, Settings } from 'lucide-react'

interface TabBarProps {
  currentTab: 'chat' | 'agents' | 'mcp' | 'settings'
  onTabChange: (tab: 'chat' | 'agents' | 'mcp' | 'settings') => void
}

export function TabBar({ currentTab, onTabChange }: TabBarProps) {
  const tabs = [
    { id: 'chat' as const, label: 'Chat', icon: MessageSquare },
    { id: 'agents' as const, label: 'Agents', icon: Boxes },
    { id: 'mcp' as const, label: 'MCP', icon: Server },
    { id: 'settings' as const, label: 'Settings', icon: Settings },
  ]

  return (
    <nav className="flex items-center gap-1">
      {tabs.map(tab => {
        const Icon = tab.icon
        const isActive = currentTab === tab.id
        
        return (
          <button
            key={tab.id}
            onClick={() => onTabChange(tab.id)}
            className={`
              flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium
              transition-all duration-200
              ${isActive 
                ? 'bg-background-tertiary text-foreground-primary' 
                : 'text-foreground-secondary hover:text-foreground-primary hover:bg-background-tertiary/50'
              }
            `}
          >
            <Icon size={16} />
            <span>{tab.label}</span>
          </button>
        )
      })}
    </nav>
  )
}