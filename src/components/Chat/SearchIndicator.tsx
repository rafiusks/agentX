import { memo, useState } from 'react'
import { Globe, ChevronDown, ChevronUp, Search, ExternalLink } from 'lucide-react'

interface SearchResult {
  title: string
  domain: string
  url: string
  summary: string
}

interface SearchMetadata {
  provider: string
  resultCount: number
  results: SearchResult[]
  searchQuery?: string
  duration?: number
}

interface SearchIndicatorProps {
  metadata?: SearchMetadata
  isSearching?: boolean
}

export const SearchIndicator = memo(function SearchIndicator({ 
  metadata, 
  isSearching 
}: SearchIndicatorProps) {
  const [isExpanded, setIsExpanded] = useState(false)

  if (isSearching) {
    return (
      <div className="flex items-center gap-2 px-3 py-2 bg-accent-blue/10 border border-accent-blue/20 rounded-lg animate-pulse">
        <Search className="w-4 h-4 text-accent-blue animate-spin" />
        <span className="text-sm text-foreground-secondary">Searching the web...</span>
      </div>
    )
  }

  if (!metadata) return null

  return (
    <div className="border border-border-subtle rounded-lg overflow-hidden">
      {/* Header */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between px-3 py-2 bg-background-secondary hover:bg-background-tertiary transition-colors"
      >
        <div className="flex items-center gap-2">
          <Globe className="w-4 h-4 text-accent-blue" />
          <span className="text-sm font-medium text-foreground-primary">
            Web Search
          </span>
          <span className="px-2 py-0.5 bg-accent-blue/10 rounded-full text-xs text-accent-blue">
            {metadata.resultCount} results
          </span>
          {metadata.duration && (
            <span className="text-xs text-foreground-muted">
              {(metadata.duration / 1000).toFixed(1)}s
            </span>
          )}
        </div>
        {isExpanded ? (
          <ChevronUp className="w-4 h-4 text-foreground-muted" />
        ) : (
          <ChevronDown className="w-4 h-4 text-foreground-muted" />
        )}
      </button>

      {/* Expanded Results */}
      {isExpanded && metadata.results && (
        <div className="px-3 py-2 space-y-2 bg-background-primary">
          <div className="text-xs text-foreground-muted">
            Provider: {metadata.provider}
            {metadata.searchQuery && (
              <> • Query: "{metadata.searchQuery}"</>
            )}
          </div>
          <div className="space-y-2">
            {metadata.results.map((result, idx) => (
              <div 
                key={idx}
                className="p-2 bg-background-secondary rounded border border-border-subtle"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1">
                    <h4 className="text-sm font-medium text-foreground-primary">
                      {idx + 1}. {result.title}
                    </h4>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs text-accent-blue">
                        {result.domain}
                      </span>
                      <a
                        href={result.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-xs text-foreground-muted hover:text-accent-blue transition-colors"
                      >
                        <ExternalLink className="w-3 h-3" />
                      </a>
                    </div>
                    <p className="text-xs text-foreground-secondary mt-1">
                      {result.summary}
                    </p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
})