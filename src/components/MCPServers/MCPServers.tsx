import { useState, useEffect } from 'react'
import { Server, Trash2 } from 'lucide-react'

export function MCPServers() {
  const [servers, _setServers] = useState<string[]>([])
  const [_tools, _setTools] = useState<string[]>([])
  const [_showAddForm, _setShowAddForm] = useState(false)
  const [_newServer, _setNewServer] = useState({
    name: '',
    command: '',
    args: ''
  })

  useEffect(() => {
    // MCP servers will be implemented in a future update
    // For now, show a placeholder
  }, [])

  return (
    <div className="flex-1 p-6">
      <div className="max-w-4xl mx-auto">
        <div className="mb-8">
          <h2 className="text-2xl font-semibold text-foreground-primary mb-2">MCP Servers</h2>
          <p className="text-foreground-secondary">
            Model Context Protocol servers extend AgentX with specialized capabilities
          </p>
        </div>

        {/* Placeholder */}
        <div className="bg-background-secondary rounded-lg border border-border-subtle p-8 text-center">
          <Server className="w-12 h-12 text-foreground-muted mx-auto mb-4" />
          <h3 className="text-lg font-medium text-foreground-primary mb-2">
            MCP Servers Coming Soon
          </h3>
          <p className="text-foreground-secondary max-w-md mx-auto">
            MCP server support will be available in a future update. This will allow you to extend AgentX with custom tools and capabilities.
          </p>
        </div>

        {/* Future implementation */}
        {servers.length > 0 && (
          <div className="space-y-4">
            {servers.map((server, index) => (
              <div
                key={index}
                className="bg-background-secondary rounded-lg border border-border-subtle p-4 flex items-center justify-between"
              >
                <div className="flex items-center gap-3">
                  <Server className="w-5 h-5 text-foreground-secondary" />
                  <div>
                    <h4 className="font-medium text-foreground-primary">{server}</h4>
                    <p className="text-sm text-foreground-muted">Running</p>
                  </div>
                </div>
                <button className="p-2 hover:bg-background-primary rounded">
                  <Trash2 className="w-4 h-4 text-foreground-secondary" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}