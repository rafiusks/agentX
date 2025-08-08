import { ProviderConnections } from '../ProviderConnections'

interface SettingsProps {
  providers: Array<{
    id: string
    name: string
    enabled: boolean
    status: string
  }>
  onProvidersUpdate: () => void
}

export function Settings({ providers: _providers, onProvidersUpdate }: SettingsProps) {
  return (
    <div className="h-full flex flex-col bg-background">
      {/* Provider Connections takes full height */}
      <ProviderConnections onClose={onProvidersUpdate} />
    </div>
  )
}