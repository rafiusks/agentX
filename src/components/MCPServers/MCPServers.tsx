import { useState, useEffect } from 'react'
import { Server, Trash2, Plus, Power, Play, AlertCircle, Check, Settings, Zap } from 'lucide-react'
import { apiClient } from '@/lib/api-client'
import { BuiltinMCPServers } from './BuiltinMCPServers'

interface MCPServer {
  id: string
  name: string
  description: string
  command: string
  args: string[]
  env: Record<string, string>
  enabled: boolean
  status: 'connected' | 'disconnected' | 'error' | 'connecting'
  last_connected_at: string | null
  tools?: MCPTool[]
  resources?: MCPResource[]
}

interface MCPTool {
  id: string
  name: string
  description: string
  enabled: boolean
  usage_count: number
}

interface MCPResource {
  id: string
  uri: string
  name: string
  description: string
  mime_type: string
}

export function MCPServers() {
  const [activeTab, setActiveTab] = useState<'builtin' | 'custom'>('builtin')
  const [servers, setServers] = useState<MCPServer[]>([])
  const [selectedServer, setSelectedServer] = useState<MCPServer | null>(null)
  const [showAddForm, setShowAddForm] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [newServer, setNewServer] = useState({
    name: '',
    description: '',
    command: '',
    args: '',
    env: '',
    enabled: true
  })

  useEffect(() => {
    fetchServers()
  }, [])

  const fetchServers = async () => {
    try {
      setLoading(true)
      const response = await apiClient.get<{ servers: MCPServer[] }>('/mcp/servers')
      setServers(response.servers || [])
      setError(null)
    } catch (err) {
      console.error('Failed to fetch MCP servers:', err)
      setError('Failed to load MCP servers')
    } finally {
      setLoading(false)
    }
  }

  const handleAddServer = async () => {
    try {
      const serverData = {
        name: newServer.name,
        description: newServer.description,
        command: newServer.command,
        args: newServer.args ? newServer.args.split(' ').filter(Boolean) : [],
        env: newServer.env ? JSON.parse(newServer.env) : {},
        enabled: newServer.enabled
      }
      
      await apiClient.post('/mcp/servers', serverData)
      await fetchServers()
      setShowAddForm(false)
      setNewServer({
        name: '',
        description: '',
        command: '',
        args: '',
        env: '',
        enabled: true
      })
    } catch (err) {
      console.error('Failed to add server:', err)
      setError('Failed to add MCP server')
    }
  }

  const handleToggleServer = async (serverId: string) => {
    try {
      await apiClient.post(`/mcp/servers/${serverId}/toggle`)
      await fetchServers()
    } catch (err) {
      console.error('Failed to toggle server:', err)
      setError('Failed to toggle MCP server')
    }
  }

  const handleDeleteServer = async (serverId: string) => {
    if (!confirm('Are you sure you want to delete this MCP server?')) return
    
    try {
      await apiClient.delete(`/mcp/servers/${serverId}`)
      await fetchServers()
      if (selectedServer?.id === serverId) {
        setSelectedServer(null)
      }
    } catch (err) {
      console.error('Failed to delete server:', err)
      setError('Failed to delete MCP server')
    }
  }

  const handleTestServer = async (serverId: string) => {
    try {
      await apiClient.post(`/mcp/servers/${serverId}/test`)
      alert('Connection test successful!')
    } catch (err) {
      console.error('Failed to test server:', err)
      alert('Connection test failed')
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'connected':
        return <Check className="w-4 h-4 text-green-500" />
      case 'connecting':
        return <div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
      case 'error':
        return <AlertCircle className="w-4 h-4 text-red-500" />
      default:
        return <div className="w-4 h-4 bg-gray-400 rounded-full" />
    }
  }

  const getStatusText = (status: string) => {
    switch (status) {
      case 'connected':
        return 'Connected'
      case 'connecting':
        return 'Connecting...'
      case 'error':
        return 'Error'
      default:
        return 'Disconnected'
    }
  }

  return (
    <div className="flex-1 p-6">
      <div className="max-w-6xl mx-auto">
        <div className="mb-8">
          <h2 className="text-2xl font-semibold text-foreground-primary mb-2">MCP Servers</h2>
          <p className="text-foreground-secondary mb-6">
            Model Context Protocol servers extend AgentX with specialized capabilities
          </p>
          
          {/* Tabs */}
          <div className="flex items-center justify-between">
            <div className="flex space-x-1 bg-background-secondary rounded-lg p-1">
              <button
                onClick={() => setActiveTab('builtin')}
                className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === 'builtin'
                    ? 'bg-primary text-white shadow-sm'
                    : 'text-foreground-secondary hover:text-foreground-primary'
                }`}
              >
                <Zap className="w-4 h-4 inline mr-2" />
                Built-in Servers
              </button>
              <button
                onClick={() => setActiveTab('custom')}
                className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === 'custom'
                    ? 'bg-primary text-white shadow-sm'
                    : 'text-foreground-secondary hover:text-foreground-primary'
                }`}
              >
                <Settings className="w-4 h-4 inline mr-2" />
                Custom Servers
              </button>
            </div>
            
            {activeTab === 'custom' && (
              <button
                onClick={() => setShowAddForm(true)}
                className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary-hover flex items-center gap-2"
              >
                <Plus className="w-4 h-4" />
                Add Server
              </button>
            )}
          </div>
        </div>

        {/* Tab Content */}
        {activeTab === 'builtin' ? (
          <BuiltinMCPServers />
        ) : (
          <>
            {error && (
              <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded-lg">
                {error}
              </div>
            )}

            {loading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        ) : servers.length === 0 ? (
          <div className="bg-background-secondary rounded-lg border border-border-subtle p-8 text-center">
            <Server className="w-12 h-12 text-foreground-muted mx-auto mb-4" />
            <h3 className="text-lg font-medium text-foreground-primary mb-2">
              No MCP Servers Configured
            </h3>
            <p className="text-foreground-secondary max-w-md mx-auto mb-4">
              Add MCP servers to extend AgentX with custom tools and capabilities.
            </p>
            <button
              onClick={() => setShowAddForm(true)}
              className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary-hover"
            >
              Add Your First Server
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Server List */}
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-foreground-primary mb-3">Configured Servers</h3>
              {servers.map((server) => (
                <div
                  key={server.id}
                  onClick={() => setSelectedServer(server)}
                  className={`bg-background-secondary rounded-lg border p-4 cursor-pointer transition-all ${
                    selectedServer?.id === server.id
                      ? 'border-primary ring-2 ring-primary/20'
                      : 'border-border-subtle hover:border-border'
                  }`}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3">
                      <Server className="w-5 h-5 text-foreground-secondary mt-1" />
                      <div className="flex-1">
                        <h4 className="font-medium text-foreground-primary">{server.name}</h4>
                        {server.description && (
                          <p className="text-sm text-foreground-muted mt-1">{server.description}</p>
                        )}
                        <div className="flex items-center gap-2 mt-2">
                          {getStatusIcon(server.status)}
                          <span className="text-sm text-foreground-secondary">
                            {getStatusText(server.status)}
                          </span>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleToggleServer(server.id)
                        }}
                        className={`p-2 rounded transition-colors ${
                          server.enabled
                            ? 'bg-green-100 dark:bg-green-900/20 text-green-600 dark:text-green-400 hover:bg-green-200 dark:hover:bg-green-900/30'
                            : 'bg-gray-100 dark:bg-gray-800 text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-700'
                        }`}
                      >
                        <Power className="w-4 h-4" />
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleTestServer(server.id)
                        }}
                        className="p-2 hover:bg-background-primary rounded transition-colors"
                      >
                        <Play className="w-4 h-4 text-foreground-secondary" />
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleDeleteServer(server.id)
                        }}
                        className="p-2 hover:bg-red-100 dark:hover:bg-red-900/20 rounded transition-colors"
                      >
                        <Trash2 className="w-4 h-4 text-red-500" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* Server Details */}
            {selectedServer && (
              <div className="bg-background-secondary rounded-lg border border-border-subtle p-6">
                <h3 className="text-lg font-medium text-foreground-primary mb-4">
                  {selectedServer.name} Details
                </h3>
                
                <div className="space-y-4">
                  <div>
                    <label className="text-sm font-medium text-foreground-secondary">Command</label>
                    <p className="mt-1 font-mono text-sm bg-background-primary rounded p-2">
                      {selectedServer.command}
                    </p>
                  </div>
                  
                  {selectedServer.args && selectedServer.args.length > 0 && (
                    <div>
                      <label className="text-sm font-medium text-foreground-secondary">Arguments</label>
                      <p className="mt-1 font-mono text-sm bg-background-primary rounded p-2">
                        {selectedServer.args.join(' ')}
                      </p>
                    </div>
                  )}
                  
                  {selectedServer.env && Object.keys(selectedServer.env).length > 0 && (
                    <div>
                      <label className="text-sm font-medium text-foreground-secondary">Environment Variables</label>
                      <div className="mt-1 font-mono text-sm bg-background-primary rounded p-2 space-y-1">
                        {Object.entries(selectedServer.env).map(([key, value]) => (
                          <div key={key}>
                            <span className="text-blue-500">{key}</span>=<span className="text-green-500">"{value}"</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                  
                  {selectedServer.tools && selectedServer.tools.length > 0 && (
                    <div>
                      <label className="text-sm font-medium text-foreground-secondary">Available Tools ({selectedServer.tools.length})</label>
                      <div className="mt-2 space-y-2">
                        {selectedServer.tools.map((tool) => (
                          <div key={tool.id} className="bg-background-primary rounded p-3">
                            <div className="flex items-center justify-between">
                              <h4 className="font-medium text-sm">{tool.name}</h4>
                              <span className="text-xs text-foreground-muted">
                                Used {tool.usage_count} times
                              </span>
                            </div>
                            {tool.description && (
                              <p className="text-xs text-foreground-secondary mt-1">{tool.description}</p>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                  
                  {selectedServer.resources && selectedServer.resources.length > 0 && (
                    <div>
                      <label className="text-sm font-medium text-foreground-secondary">Available Resources ({selectedServer.resources.length})</label>
                      <div className="mt-2 space-y-2">
                        {selectedServer.resources.map((resource) => (
                          <div key={resource.id} className="bg-background-primary rounded p-3">
                            <h4 className="font-medium text-sm">{resource.name}</h4>
                            <p className="text-xs text-foreground-muted mt-1">{resource.uri}</p>
                            {resource.description && (
                              <p className="text-xs text-foreground-secondary mt-1">{resource.description}</p>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

            {/* Add Server Modal */}
            {showAddForm && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-background-primary rounded-lg p-6 w-full max-w-md">
              <h3 className="text-lg font-medium text-foreground-primary mb-4">Add MCP Server</h3>
              
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-foreground-secondary mb-1">
                    Name <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={newServer.name}
                    onChange={(e) => setNewServer({ ...newServer, name: e.target.value })}
                    className="w-full px-3 py-2 bg-background-secondary border border-border-subtle rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                    placeholder="e.g., Code RAG"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-foreground-secondary mb-1">
                    Description
                  </label>
                  <input
                    type="text"
                    value={newServer.description}
                    onChange={(e) => setNewServer({ ...newServer, description: e.target.value })}
                    className="w-full px-3 py-2 bg-background-secondary border border-border-subtle rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                    placeholder="e.g., Semantic code search and analysis"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-foreground-secondary mb-1">
                    Command <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={newServer.command}
                    onChange={(e) => setNewServer({ ...newServer, command: e.target.value })}
                    className="w-full px-3 py-2 bg-background-secondary border border-border-subtle rounded-lg focus:outline-none focus:ring-2 focus:ring-primary font-mono text-sm"
                    placeholder="e.g., /usr/local/bin/code-rag"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-foreground-secondary mb-1">
                    Arguments (space-separated)
                  </label>
                  <input
                    type="text"
                    value={newServer.args}
                    onChange={(e) => setNewServer({ ...newServer, args: e.target.value })}
                    className="w-full px-3 py-2 bg-background-secondary border border-border-subtle rounded-lg focus:outline-none focus:ring-2 focus:ring-primary font-mono text-sm"
                    placeholder="e.g., mcp-server --verbose"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-foreground-secondary mb-1">
                    Environment Variables (JSON)
                  </label>
                  <textarea
                    value={newServer.env}
                    onChange={(e) => setNewServer({ ...newServer, env: e.target.value })}
                    className="w-full px-3 py-2 bg-background-secondary border border-border-subtle rounded-lg focus:outline-none focus:ring-2 focus:ring-primary font-mono text-sm"
                    rows={3}
                    placeholder='{"DATABASE_URL": "postgresql://..."}'
                  />
                </div>
                
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="enabled"
                    checked={newServer.enabled}
                    onChange={(e) => setNewServer({ ...newServer, enabled: e.target.checked })}
                    className="rounded"
                  />
                  <label htmlFor="enabled" className="text-sm text-foreground-secondary">
                    Enable server immediately
                  </label>
                </div>
              </div>
              
              <div className="flex justify-end gap-3 mt-6">
                <button
                  onClick={() => {
                    setShowAddForm(false)
                    setNewServer({
                      name: '',
                      description: '',
                      command: '',
                      args: '',
                      env: '',
                      enabled: true
                    })
                  }}
                  className="px-4 py-2 text-foreground-secondary hover:text-foreground-primary"
                >
                  Cancel
                </button>
                <button
                  onClick={handleAddServer}
                  disabled={!newServer.name || !newServer.command}
                  className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary-hover disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Add Server
                </button>
              </div>
            </div>
          </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}