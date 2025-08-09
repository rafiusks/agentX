import { useState, useEffect } from 'react'
import { Chat } from './components/Chat/Chat'
import { ConnectionSelector } from './components/ConnectionSelector/ConnectionSelector'
import { NavigationSidebar } from './components/Navigation/NavigationSidebar'
import { useNavigationStore } from './stores/navigation.store'
import { Settings } from './components/Settings/Settings'
import { Welcome } from './components/Welcome/Welcome'
import { Help } from './components/Help/Help'
import { MCPServers } from './components/MCPServers/MCPServers'
import { useChatStore } from './stores/chat.store'
import { useCanvasStore } from './stores/canvas.store'
import { Button } from './components/ui/button'
import { SplitSquareHorizontal } from 'lucide-react'
import { FEATURES } from './config/features'

function App() {
  const [currentTab, setCurrentTab] = useState<'chat' | 'agents' | 'mcp' | 'settings'>('chat')
  const [showWelcome, setShowWelcome] = useState(false)
  const [showHelp, setShowHelp] = useState(false)
  
  const { currentConnectionId, setCurrentConnectionId, createSession } = useChatStore()
  const { isCanvasOpen, toggleCanvas } = useCanvasStore()
  const { isSidebarCollapsed, toggleSidebar } = useNavigationStore();

  useEffect(() => {
    // Check if this is first run
    const welcomeCompleted = localStorage.getItem('agentx-welcome-completed')
    if (!welcomeCompleted) {
      setShowWelcome(true)
    }

    // Listen for keyboard shortcuts
    const handleKeyDown = (e: KeyboardEvent) => {
      
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
      
      // Toggle sidebar
      if ((e.metaKey || e.ctrlKey) && e.key === '/') {
        e.preventDefault()
        toggleSidebar()
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
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [showHelp, toggleSidebar]) // Remove loadConnections and createSession from deps to avoid re-renders
  
  if (showWelcome) {
    return <Welcome onComplete={() => {
      setShowWelcome(false)
    }} />
  }
  
  return (
    <div className="flex h-screen bg-background-primary text-foreground-primary" style={{ backgroundColor: '#0a0a0a', color: 'white', minHeight: '100vh' }}>
      {/* Sidebar Navigation */}
      <NavigationSidebar 
        currentTab={currentTab}
        onTabChange={setCurrentTab}
        isCollapsed={isSidebarCollapsed}
        onToggleCollapse={toggleSidebar}
      />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Slim Header Bar */}
        <header className="flex items-center justify-between h-10 px-4 border-b border-border-subtle/50 bg-background-primary/80 backdrop-blur-sm">
          <div className="flex items-center gap-3">
            <ConnectionSelector 
              currentConnectionId={currentConnectionId}
              onConnectionChange={setCurrentConnectionId}
            />
          </div>
          
          <div className="flex items-center gap-3">
            {FEATURES.CANVAS_MODE && currentTab === 'chat' && (
              <Button
                variant={isCanvasOpen ? "default" : "ghost"}
                size="sm"
                onClick={toggleCanvas}
                title="Toggle Canvas"
              >
                <SplitSquareHorizontal size={16} />
                <span className="ml-2">Canvas</span>
              </Button>
            )}
          </div>
        </header>
        
        {/* Content */}
        <main className="flex-1 flex overflow-hidden">
        {currentTab === 'chat' && (
          <Chat />
        )}
        
        {currentTab === 'agents' && (
          <div className="flex items-center justify-center h-full">
            <div className="text-center space-y-4">
              <h2 className="text-title-2 text-foreground-secondary">
                Agent Orchestration
              </h2>
              <p className="text-foreground-muted">
                Coming soon: Visual agent workflow builder
              </p>
            </div>
          </div>
        )}

        {currentTab === 'mcp' && (
          <div className="p-6">
            <MCPServers />
          </div>
        )}
        
        {currentTab === 'settings' && (
          <div className="overflow-auto h-full">
            <Settings 
              providers={[]}
              onProvidersUpdate={() => {}}
            />
          </div>
        )}
        </main>
      </div>

      {/* Help Dialog */}
      <Help open={showHelp} onClose={() => setShowHelp(false)} />
    </div>
  )
}

export default App