import { memo } from 'react'
import { Globe, Search, Info } from 'lucide-react'
import { usePreferencesStore } from '../../stores/preferences.store'

interface SearchToggleProps {
  isCompact?: boolean
}

export const SearchToggle = memo(function SearchToggle({ isCompact = false }: SearchToggleProps) {
  const { forceWebSearch, setForceWebSearch } = usePreferencesStore()

  if (isCompact) {
    return (
      <button
        onClick={() => setForceWebSearch(!forceWebSearch)}
        className={`
          p-2 rounded-lg transition-all
          ${forceWebSearch 
            ? 'bg-accent-blue/20 text-accent-blue border border-accent-blue/30' 
            : 'bg-background-secondary text-foreground-muted hover:text-foreground-secondary border border-border-subtle'
          }
        `}
        title={forceWebSearch ? "Web search enabled for all queries" : "Click to enable web search for all queries"}
      >
        <Globe className="w-4 h-4" />
      </button>
    )
  }

  return (
    <div className="flex items-center gap-3 p-2 rounded-lg border border-border-subtle bg-background-secondary">
      <button
        onClick={() => setForceWebSearch(!forceWebSearch)}
        className={`
          flex items-center gap-2 px-3 py-1.5 rounded-md transition-all font-medium text-sm
          ${forceWebSearch 
            ? 'bg-accent-blue text-white' 
            : 'bg-background-tertiary text-foreground-secondary hover:text-foreground-primary'
          }
        `}
      >
        {forceWebSearch ? (
          <>
            <Search className="w-4 h-4" />
            <span>Search ON</span>
          </>
        ) : (
          <>
            <Globe className="w-4 h-4" />
            <span>Search OFF</span>
          </>
        )}
      </button>
      
      <div className="flex items-center gap-1">
        <Info className="w-3 h-3 text-foreground-muted" />
        <span className="text-xs text-foreground-muted">
          {forceWebSearch 
            ? "Searching web for all queries" 
            : "Web search only when needed"
          }
        </span>
      </div>
    </div>
  )
})