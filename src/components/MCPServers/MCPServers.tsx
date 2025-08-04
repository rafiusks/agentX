import { useState, useEffect } from 'react'
import { invoke } from '@tauri-apps/api/core'
import { Plus, Server, Trash2, Wrench } from 'lucide-react'

export function MCPServers() {
  const [servers, setServers] = useState<string[]>([])
  const [tools, setTools] = useState<string[]>([])
  const [showAddForm, setShowAddForm] = useState(false)
  const [newServer, setNewServer] = useState({
    name: '',
    command: '',
    args: ''
  })

  useEffect(() => {
    loadServers()
    loadTools()
  }, [])

  const loadServers = async () => {
    try {
      const serverList = await invoke<string[]>('get_mcp_servers')
      setServers(serverList)
    } catch (error) {
      console.error('Failed to load MCP servers:', error)
    }
  }

  const loadTools = async () => {
    try {
      const toolList = await invoke<string[]>('list_mcp_tools')
      setTools(toolList)
    } catch (error) {
      console.error('Failed to load MCP tools:', error)
    }
  }

  const handleAddServer = async () => {
    console.log('handleAddServer called', { newServer })
    
    if (!newServer.name || !newServer.command) {
      console.log('Missing name or command', { name: newServer.name, command: newServer.command })
      return
    }

    try {
      const args = newServer.args ? newServer.args.split(' ').filter(arg => arg.trim()) : []
      console.log('Calling add_mcp_server with:', { name: newServer.name, command: newServer.command, args })
      
      await invoke('add_mcp_server', {
        name: newServer.name,
        command: newServer.command,
        args
      })

      console.log('MCP server added successfully')
      setNewServer({ name: '', command: '', args: '' })
      setShowAddForm(false)
      await loadServers()
      await loadTools()
    } catch (error) {
      console.error('Failed to add MCP server:', error)
      alert(`Failed to add MCP server: ${error}`)
    }
  }

  const handleDeleteServer = async (serverName: string) => {
    try {
      console.log('Deleting MCP server:', serverName)
      await invoke('remove_mcp_server', { name: serverName })
      console.log('MCP server deleted successfully')
      await loadServers()
      await loadTools()
    } catch (error) {
      console.error('Failed to delete MCP server:', error)
      alert(`Failed to delete MCP server: ${error}`)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium text-foreground">MCP Servers</h3>
          <p className="text-sm text-foreground-muted">
            Connect to Model Context Protocol servers for enhanced AI capabilities
          </p>
        </div>
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className="btn-primary flex items-center gap-2"
        >
          <Plus size={16} />
          Add Server
        </button>
      </div>

      {showAddForm && (
        <div className="card space-y-4">
          <h4 className="font-medium">Add MCP Server</h4>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Name</label>
              <input
                type="text"
                value={newServer.name}
                onChange={(e) => setNewServer(prev => ({ ...prev, name: e.target.value }))}
                placeholder="My MCP Server"
                className="input"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Command</label>
              <input
                type="text"
                value={newServer.command}
                onChange={(e) => setNewServer(prev => ({ ...prev, command: e.target.value }))}
                placeholder="python3 server.py"
                className="input"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Arguments</label>
              <input
                type="text"
                value={newServer.args}
                onChange={(e) => setNewServer(prev => ({ ...prev, args: e.target.value }))}
                placeholder="--port 8080"
                className="input"
              />
            </div>
          </div>
          <div className="flex gap-2">
            <button onClick={handleAddServer} className="btn-primary">
              Add Server
            </button>
            <button 
              onClick={() => setShowAddForm(false)}
              className="btn-secondary"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Connected Servers */}
        <div className="card">
          <div className="flex items-center gap-2 mb-4">
            <Server size={20} className="text-accent" />
            <h4 className="font-medium">Connected Servers</h4>
          </div>
          
          {servers.length === 0 ? (
            <p className="text-foreground-muted text-sm">No MCP servers connected</p>
          ) : (
            <div className="space-y-2">
              {servers.map((server, index) => (
                <div key={index} className="flex items-center justify-between p-3 bg-background-secondary rounded-lg">
                  <div className="flex items-center gap-3">
                    <div className="w-2 h-2 bg-accent-green rounded-full" />
                    <span className="font-mono text-sm">{server}</span>
                  </div>
                  <button 
                    onClick={() => handleDeleteServer(server)}
                    className="text-foreground-muted hover:text-red-500 transition-colors"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Available Tools */}
        <div className="card">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <Wrench size={20} className="text-accent" />
              <h4 className="font-medium">Available Tools</h4>
            </div>
            <button 
              onClick={() => { loadTools(); console.log('Refreshing tools...'); }}
              className="text-xs text-accent hover:text-accent-bright px-2 py-1 rounded bg-background-secondary hover:bg-background-tertiary transition-colors"
            >
              Refresh
            </button>
          </div>
          
          {tools.length === 0 ? (
            <p className="text-foreground-muted text-sm">No tools available</p>
          ) : (
            <div className="space-y-1 max-h-64 overflow-y-auto">
              {tools.map((tool, index) => (
                <div key={index} className="p-2 bg-background-secondary rounded text-xs font-mono">
                  {tool}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Example Servers */}
      <div className="card">
        <h4 className="font-medium mb-3">Example MCP Servers</h4>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
          <div className="p-3 bg-background-secondary rounded-lg">
            <div className="font-medium mb-1">File System</div>
            <div className="text-foreground-muted mb-2">Read and write files</div>
            <code className="text-xs">npx @modelcontextprotocol/server-filesystem /path/to/directory</code>
          </div>
          <div className="p-3 bg-background-secondary rounded-lg">
            <div className="font-medium mb-1">Git</div>
            <div className="text-foreground-muted mb-2">Git repository operations</div>
            <code className="text-xs">npx @modelcontextprotocol/server-git</code>
          </div>
          <div className="p-3 bg-background-secondary rounded-lg">
            <div className="font-medium mb-1">PostgreSQL</div>
            <div className="text-foreground-muted mb-2">Database queries</div>
            <code className="text-xs">npx @modelcontextprotocol/server-postgres</code>
          </div>
          <div className="p-3 bg-background-secondary rounded-lg">
            <div className="font-medium mb-1">Brave Search</div>
            <div className="text-foreground-muted mb-2">Web search capabilities</div>
            <code className="text-xs">npx @modelcontextprotocol/server-brave-search</code>
          </div>
        </div>
      </div>
    </div>
  )
}