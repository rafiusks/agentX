import { memo } from 'react'
import { Globe, Info } from 'lucide-react'
import { usePreferencesStore } from '../../stores/preferences.store'

export const SearchSettings = memo(function SearchSettings() {
  const { searchMode, setSearchMode } = usePreferencesStore()

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-2">
        <Globe className="w-5 h-5 text-accent-blue" />
        <h3 className="text-lg font-semibold">Web Search Settings</h3>
      </div>

      <div className="space-y-3">
        <label className="block">
          <span className="text-sm font-medium text-foreground-primary mb-2 block">
            Search Mode
          </span>
          <select
            value={searchMode || 'balanced'}
            onChange={(e) => setSearchMode(e.target.value as 'conservative' | 'balanced' | 'aggressive')}
            className="w-full px-3 py-2 bg-background-secondary border border-border-subtle rounded-lg focus:outline-none focus:ring-2 focus:ring-accent-blue"
          >
            <option value="conservative">Conservative - Only search when explicitly asked</option>
            <option value="balanced">Balanced - Search for likely outdated info (default)</option>
            <option value="aggressive">Aggressive - Search for most queries to ensure accuracy</option>
          </select>
        </label>

        <div className="flex items-start gap-2 p-3 bg-accent-blue/5 border border-accent-blue/20 rounded-lg">
          <Info className="w-4 h-4 text-accent-blue mt-0.5" />
          <div className="text-sm text-foreground-secondary">
            <p className="font-medium text-foreground-primary mb-1">How search modes work:</p>
            <ul className="space-y-1 ml-4">
              <li>• <strong>Conservative:</strong> Only searches when you use keywords like "search", "look up", etc.</li>
              <li>• <strong>Balanced:</strong> Automatically searches for AI models, recent events, and time-sensitive topics</li>
              <li>• <strong>Aggressive:</strong> Searches for most questions to ensure you get the latest information</li>
            </ul>
          </div>
        </div>

        <div className="p-3 bg-background-tertiary rounded-lg">
          <p className="text-sm text-foreground-secondary">
            💡 <strong>Tip:</strong> Use Aggressive mode when accuracy is critical or when discussing recent events. 
            Web searches add ~2-3 seconds to response time.
          </p>
        </div>
      </div>
    </div>
  )
})