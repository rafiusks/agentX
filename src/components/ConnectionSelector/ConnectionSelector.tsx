import { ChevronDown, Circle, Settings } from 'lucide-react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { useEffect, useState } from 'react'
import { api } from '@/services/api'

interface Connection {
  id: string
  provider_id: string
  name: string
  enabled: boolean
  status: string
}

interface ConnectionSelectorProps {
  currentConnectionId: string | null
  onConnectionChange: (connectionId: string) => void
}

const providerIcons: Record<string, string> = {
  openai: '⚡',
  anthropic: '🧠',
  ollama: '🦙',
  'lm-studio': '🖥️',
}

const statusColors: Record<string, string> = {
  connected: 'text-accent-green',
  connecting: 'text-accent-yellow',
  error: 'text-accent-red',
  disconnected: 'text-foreground-muted',
  disabled: 'text-foreground-tertiary',
}

export function ConnectionSelector({ currentConnectionId, onConnectionChange }: ConnectionSelectorProps) {
  const [connections, setConnections] = useState<Connection[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadConnections()
  }, [])

  const loadConnections = async () => {
    try {
      const result = await api.listConnections()
      setConnections(result)
      
      // Set default connection if none selected
      if (!currentConnectionId && result.length > 0) {
        const enabledConnection = result.find((c: any) => c.enabled) || result[0]
        if (enabledConnection) {
          onConnectionChange(enabledConnection.id)
        }
      }
    } catch (error) {
      console.error('Failed to load connections:', error)
    } finally {
      setLoading(false)
    }
  }

  const current = connections.find(c => c.id === currentConnectionId)
  const hasConnections = connections.length > 0

  // Group connections by provider
  const groupedConnections = connections.reduce((acc, conn) => {
    if (!acc[conn.provider_id]) {
      acc[conn.provider_id] = []
    }
    acc[conn.provider_id].push(conn)
    return acc
  }, {} as Record<string, Connection[]>)

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button className="btn-secondary px-4 py-2 gap-3 min-w-[180px] justify-between">
          <div className="flex items-center gap-2">
            {current ? (
              <>
                <Circle 
                  size={8} 
                  className={`fill-current ${statusColors[current.status]}`}
                />
                <span className="text-sm">
                  {current.name}
                </span>
              </>
            ) : (
              <span className="text-sm text-foreground-muted">
                {loading ? 'Loading...' : 'No Connection'}
              </span>
            )}
          </div>
          <ChevronDown size={14} className="text-foreground-secondary" />
        </button>
      </DropdownMenu.Trigger>

      <DropdownMenu.Portal>
        <DropdownMenu.Content
          className="min-w-[240px] bg-background-secondary rounded-md p-1 shadow-xl border border-border-subtle animate-in"
          sideOffset={5}
        >
          {hasConnections ? (
            <>
              {Object.entries(groupedConnections).map(([providerId, conns]) => (
                <div key={providerId}>
                  <div className="px-3 py-1.5 text-xs text-foreground-muted flex items-center gap-2">
                    <span>{providerIcons[providerId]}</span>
                    <span>{providerId.charAt(0).toUpperCase() + providerId.slice(1)}</span>
                  </div>
                  {conns.map(connection => (
                    <DropdownMenu.Item
                      key={connection.id}
                      className={`
                        flex items-center justify-between px-3 py-2.5 rounded-md text-sm
                        outline-none cursor-pointer transition-colors
                        ${connection.enabled 
                          ? 'hover:bg-background-tertiary text-foreground-primary' 
                          : 'text-foreground-muted opacity-60'
                        }
                        ${currentConnectionId === connection.id ? 'bg-background-tertiary' : ''}
                      `}
                      onSelect={() => connection.enabled && onConnectionChange(connection.id)}
                      disabled={!connection.enabled}
                    >
                      <div className="flex items-center gap-2">
                        <Circle 
                          size={8} 
                          className={`fill-current ${statusColors[connection.status]}`}
                        />
                        <span>{connection.name}</span>
                      </div>
                      <span className="text-xs text-foreground-muted">
                        {connection.status}
                      </span>
                    </DropdownMenu.Item>
                  ))}
                </div>
              ))}
            </>
          ) : (
            <div className="px-3 py-4 text-center">
              <p className="text-sm text-foreground-muted mb-2">No connections configured</p>
            </div>
          )}
          
          <DropdownMenu.Separator className="h-px bg-border-subtle my-1" />
          
          <DropdownMenu.Item
            className="px-3 py-2.5 text-sm flex items-center gap-2 hover:bg-background-tertiary rounded-md outline-none cursor-pointer"
            onSelect={() => {
              // Navigate to settings
              if (window.location.pathname !== '/settings') {
                window.location.pathname = '/settings'
              }
            }}
          >
            <Settings size={14} />
            <span>Manage Connections</span>
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}