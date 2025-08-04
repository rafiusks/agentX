import { useState, useEffect } from 'react'
import { invoke } from '@tauri-apps/api/core'
import { Chat } from './components/Chat/Chat'
import { ModelSelector } from './components/ModelSelector/ModelSelector'
import { CommandPalette } from './components/CommandPalette/CommandPalette'
import { TabBar } from './components/TabBar/TabBar'
import { Settings } from './components/Settings/Settings'
import { Welcome } from './components/Welcome/Welcome'
import { Help } from './components/Help/Help'
import { MCPServers } from './components/MCPServers/MCPServers'
import { ModeToggle } from './components/ModeToggle/ModeToggle'
import { useChatStore } from './stores/chat.store'
import { useUIStore } from './stores/ui.store'
import { Command } from 'lucide-react'

interface ProviderInfo {
  id: string
  name: string
  enabled: boolean
  status: string
}

function App() {
  console.log('App component rendering...');
  
  const [providers, setProviders] = useState<ProviderInfo[]>([])
  const [currentTab, setCurrentTab] = useState<'chat' | 'agents' | 'mcp' | 'settings'>('chat')
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false)
  const [showWelcome, setShowWelcome] = useState(false)
  const [showHelp, setShowHelp] = useState(false)
  // const [settingsOpen, setSettingsOpen] = useState(false)
  
  const { currentProvider, setCurrentProvider, createSession } = useChatStore()
  const { mode } = useUIStore()

  useEffect(() => {
    // Check if this is first run
    const welcomeCompleted = localStorage.getItem('agentx-welcome-completed')
    if (!welcomeCompleted) {
      setShowWelcome(true)
    }
    
    // Load providers on startup
    loadProviders()

    // Listen for keyboard shortcuts
    const handleKeyDown = (e: KeyboardEvent) => {
      // Command palette
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setCommandPaletteOpen(true)
      }
      
      // Help
      if (e.key === 'F1') {
        e.preventDefault()
        setShowHelp(true)
      }
      
      // New chat
      if ((e.metaKey || e.ctrlKey) && e.key === 'n') {
        e.preventDefault()
        createSession()
        setCurrentTab('chat')
      }
      
      // Settings
      if ((e.metaKey || e.ctrlKey) && e.key === ',') {
        e.preventDefault()
        setCurrentTab('settings')
      }
      
      // Tab switching
      if (e.metaKey || e.ctrlKey) {
        if (e.key === '1') {
          e.preventDefault()
          setCurrentTab('chat')
        } else if (e.key === '2') {
          e.preventDefault()
          setCurrentTab('agents')
        } else if (e.key === '3') {
          e.preventDefault()
          setCurrentTab('settings')
        }
      }
      
      // Close dialogs on Escape
      if (e.key === 'Escape') {
        if (showHelp) {
          e.preventDefault()
          setShowHelp(false)
        } else if (commandPaletteOpen) {
          e.preventDefault()
          setCommandPaletteOpen(false)
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [showHelp, commandPaletteOpen, createSession])

  const loadProviders = async () => {
    try {
      const result = await invoke<ProviderInfo[]>('get_providers')
      setProviders(result)
      
      // Set default provider if none selected
      if (!currentProvider && result.length > 0) {
        const enabledProvider = result.find(p => p.enabled) || result[0]
        setCurrentProvider(enabledProvider.id)
      }
    } catch (error) {
      console.error('Failed to load providers:', error)
    }
  }

  console.log('App render - providers:', providers.length, 'currentProvider:', currentProvider);
  
  if (showWelcome) {
    return <Welcome onComplete={() => {
      setShowWelcome(false)
      loadProviders() // Reload providers after setup
    }} />
  }
  
  return (
    <div className="flex flex-col h-screen bg-background-primary text-foreground-primary" style={{ backgroundColor: '#0a0a0a', color: 'white', minHeight: '100vh' }}>
      {/* Header */}
      <header className="flex items-center justify-between px-6 py-4 border-b border-border-subtle glass">
        <div className="flex items-center gap-4">
          <h1 className="text-xl font-semibold">AgentX</h1>
          {mode !== 'simple' && (
            <TabBar currentTab={currentTab} onTabChange={setCurrentTab} />
          )}
        </div>
        
        <div className="flex items-center gap-3">
          <ModeToggle />
          
          <div className="w-px h-6 bg-border-subtle" />
          
          <ModelSelector 
            providers={providers} 
            currentProvider={currentProvider}
            onProviderChange={setCurrentProvider}
          />
          
          {mode !== 'simple' && (
            <button
              onClick={() => setCommandPaletteOpen(true)}
              className="btn-secondary px-3 py-2 gap-2"
            >
              <Command size={14} />
              <span className="text-xs text-foreground-secondary">⌘K</span>
            </button>
          )}
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 overflow-hidden">
        {currentTab === 'chat' && (
          <Chat />
        )}
        
        {currentTab === 'agents' && (
          <div className="flex items-center justify-center h-full">
            <div className="text-center space-y-4">
              <h2 className="text-2xl font-semibold text-foreground-secondary">
                Agent Orchestration
              </h2>
              <p className="text-foreground-muted">
                Coming soon: Visual agent workflow builder
              </p>
            </div>
          </div>
        )}

        {currentTab === 'mcp' && (
          <div className="p-6 overflow-auto">
            <MCPServers />
          </div>
        )}
        
        {currentTab === 'settings' && (
          <Settings 
            providers={providers}
            onProvidersUpdate={loadProviders}
          />
        )}
      </main>

      {/* Command Palette */}
      <CommandPalette 
        open={commandPaletteOpen}
        onOpenChange={setCommandPaletteOpen}
        onSettingsOpen={() => {
          // setSettingsOpen(true)
          setCurrentTab('settings')
        }}
      />
      
      {/* Help Dialog */}
      <Help open={showHelp} onClose={() => setShowHelp(false)} />
    </div>
  )
}

export default App