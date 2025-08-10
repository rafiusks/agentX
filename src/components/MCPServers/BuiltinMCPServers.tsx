import { useState, useEffect } from 'react'
import { Server, ToggleLeft, ToggleRight, ExternalLink, Settings, Check, AlertCircle } from 'lucide-react'
import { apiClient } from '@/lib/api-client'

interface BuiltinMCPServer {
  id: string
  name: string
  description: string
  version: string
  category: string
  enabled: boolean
  tools: Array<{
    name: string
    description: string
    example?: any
  }>
  config: Record<string, any>
  required: string[]
}

export function BuiltinMCPServers() {
  const [servers, setServers] = useState<BuiltinMCPServer[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [toggling, setToggling] = useState<string | null>(null)
  const [selectedServer, setSelectedServer] = useState<BuiltinMCPServer | null>(null)

  useEffect(() => {
    fetchBuiltinServers()
  }, [])

  const fetchBuiltinServers = async () => {
    try {
      setLoading(true)
      const response = await apiClient.get<{ servers: BuiltinMCPServer[] }>('/mcp/builtin/user')
      setServers(response.servers || [])
      setError(null)
    } catch (err) {
      console.error('Failed to fetch built-in MCP servers:', err)
      setError('Failed to load built-in MCP servers')
    } finally {
      setLoading(false)
    }
  }

  const handleToggleServer = async (serverId: string, enabled: boolean) => {
    try {
      setToggling(serverId)
      await apiClient.post(`/mcp/builtin/${serverId}/toggle`, { enabled })
      
      // Update local state
      setServers(prev => prev.map(server => 
        server.id === serverId ? { ...server, enabled } : server
      ))
      
      setError(null)
    } catch (err) {
      console.error(`Failed to ${enabled ? 'enable' : 'disable'} server:`, err)
      setError(`Failed to ${enabled ? 'enable' : 'disable'} ${serverId}`)
    } finally {
      setToggling(null)
    }
  }

  const handleConvertToRegular = async (serverId: string) => {
    if (!confirm('This will create a regular MCP server that you can customize. The built-in version will be disabled. Continue?')) {
      return
    }

    try {
      await apiClient.post(`/mcp/builtin/${serverId}/convert`)
      await fetchBuiltinServers()
      alert('Successfully converted to regular MCP server! Check the MCP Servers tab.')
    } catch (err) {
      console.error('Failed to convert server:', err)
      setError('Failed to convert server to regular MCP server')
    }
  }

  const checkRequirements = (server: BuiltinMCPServer): { satisfied: boolean; missing: string[] } => {
    // This is a simple check - in a real implementation, you'd want to verify these on the backend
    const missing: string[] = []
    
    // For now, assume Node.js is always available if we're running the app
    // In the future, this could be enhanced with actual system checks
    
    return {
      satisfied: missing.length === 0,
      missing
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="mb-8">
        <h3 className="text-lg font-semibold text-foreground-primary mb-2">Built-in MCP Servers</h3>
        <p className="text-foreground-secondary">
          Pre-configured MCP servers that come with AgentX. Simply toggle them on to start using their capabilities.
        </p>
      </div>

      {error && (
        <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded-lg">
          {error}
        </div>
      )}

      {servers.length === 0 ? (
        <div className="bg-background-secondary rounded-lg border border-border-subtle p-8 text-center">
          <Server className="w-12 h-12 text-foreground-muted mx-auto mb-4" />
          <h4 className="text-lg font-medium text-foreground-primary mb-2">
            No Built-in Servers Available
          </h4>
          <p className="text-foreground-secondary">
            Built-in MCP servers will appear here when available.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Server List */}
          <div className="space-y-4">
            {servers.map((server) => {
              const requirements = checkRequirements(server)
              const canEnable = requirements.satisfied
              
              return (
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
                    <div className="flex items-start gap-3 flex-1">
                      <Server className="w-5 h-5 text-foreground-secondary mt-1" />
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <h4 className="font-medium text-foreground-primary">{server.name}</h4>
                          <span className="text-xs bg-primary/10 text-primary px-2 py-1 rounded-full">
                            {server.category}
                          </span>
                        </div>
                        <p className="text-sm text-foreground-muted mb-2">{server.description}</p>
                        <div className="flex items-center gap-4 text-xs text-foreground-secondary">
                          <span>v{server.version}</span>
                          <span>{server.tools.length} tools</span>
                          {!requirements.satisfied && (
                            <span className="flex items-center gap-1 text-orange-600">
                              <AlertCircle className="w-3 h-3" />
                              Missing dependencies
                            </span>
                          )}
                        </div>
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-2">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleToggleServer(server.id, !server.enabled)
                        }}
                        disabled={toggling === server.id || (!server.enabled && !canEnable)}
                        className={`p-2 rounded transition-colors ${
                          toggling === server.id
                            ? 'opacity-50 cursor-not-allowed'
                            : server.enabled
                            ? 'bg-green-100 dark:bg-green-900/20 text-green-600 dark:text-green-400 hover:bg-green-200 dark:hover:bg-green-900/30'
                            : canEnable
                            ? 'bg-gray-100 dark:bg-gray-800 text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-700'
                            : 'bg-gray-50 dark:bg-gray-900/50 text-gray-400 cursor-not-allowed'
                        }`}
                        title={
                          !canEnable 
                            ? `Missing dependencies: ${requirements.missing.join(', ')}`
                            : server.enabled 
                            ? 'Disable server' 
                            : 'Enable server'
                        }
                      >
                        {toggling === server.id ? (
                          <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
                        ) : server.enabled ? (
                          <ToggleRight className="w-4 h-4" />
                        ) : (
                          <ToggleLeft className="w-4 h-4" />
                        )}
                      </button>
                      
                      {server.enabled && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            handleConvertToRegular(server.id)
                          }}
                          className="p-2 hover:bg-background-primary rounded transition-colors"
                          title="Convert to regular MCP server for customization"
                        >
                          <ExternalLink className="w-4 h-4 text-foreground-secondary" />
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              )
            })}
          </div>

          {/* Server Details */}
          {selectedServer && (
            <div className="bg-background-secondary rounded-lg border border-border-subtle p-6">
              <h4 className="text-lg font-medium text-foreground-primary mb-4">
                {selectedServer.name} Details
              </h4>
              
              <div className="space-y-4">
                {/* Status */}
                <div>
                  <label className="text-sm font-medium text-foreground-secondary">Status</label>
                  <div className="mt-1 flex items-center gap-2">
                    {selectedServer.enabled ? (
                      <>
                        <Check className="w-4 h-4 text-green-500" />
                        <span className="text-green-600 dark:text-green-400">Enabled</span>
                      </>
                    ) : (
                      <>
                        <AlertCircle className="w-4 h-4 text-gray-500" />
                        <span className="text-gray-500">Disabled</span>
                      </>
                    )}
                  </div>
                </div>

                {/* Requirements */}
                {selectedServer.required.length > 0 && (
                  <div>
                    <label className="text-sm font-medium text-foreground-secondary">Requirements</label>
                    <div className="mt-1">
                      {selectedServer.required.map((req, index) => (
                        <div key={index} className="flex items-center gap-2 text-sm">
                          <Check className="w-3 h-3 text-green-500" />
                          <span>{req}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                
                {/* Available Tools */}
                <div>
                  <label className="text-sm font-medium text-foreground-secondary">
                    Available Tools ({selectedServer.tools.length})
                  </label>
                  <div className="mt-2 space-y-2">
                    {selectedServer.tools.map((tool, index) => (
                      <div key={index} className="bg-background-primary rounded p-3">
                        <div className="flex items-center justify-between">
                          <h5 className="font-medium text-sm">{tool.name}</h5>
                        </div>
                        <p className="text-xs text-foreground-secondary mt-1">{tool.description}</p>
                        {tool.example && (
                          <details className="mt-2">
                            <summary className="text-xs text-primary cursor-pointer">Show example</summary>
                            <pre className="text-xs text-foreground-muted mt-1 overflow-x-auto">
                              {JSON.stringify(tool.example, null, 2)}
                            </pre>
                          </details>
                        )}
                      </div>
                    ))}
                  </div>
                </div>

                {/* Configuration */}
                {Object.keys(selectedServer.config).length > 0 && (
                  <div>
                    <label className="text-sm font-medium text-foreground-secondary">Configuration</label>
                    <div className="mt-2 bg-background-primary rounded p-3">
                      <div className="space-y-1 text-xs font-mono">
                        {Object.entries(selectedServer.config).map(([key, value]) => (
                          <div key={key}>
                            <span className="text-blue-500">{key}</span>=
                            <span className="text-green-500">"{value}"</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                )}

                {/* Actions */}
                {selectedServer.enabled && (
                  <div className="pt-4 border-t border-border-subtle">
                    <button
                      onClick={() => handleConvertToRegular(selectedServer.id)}
                      className="w-full px-4 py-2 bg-primary/10 text-primary hover:bg-primary/20 rounded-lg transition-colors flex items-center justify-center gap-2"
                    >
                      <ExternalLink className="w-4 h-4" />
                      Convert to Regular MCP Server
                    </button>
                    <p className="text-xs text-foreground-muted mt-2 text-center">
                      Create a customizable copy that you can modify and configure
                    </p>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}