import { memo } from 'react'
import { ExternalLink } from 'lucide-react'

interface Source {
  id: string
  domain: string
  url?: string
  title?: string
}

interface SourceCitationsProps {
  sources: Source[]
}

export const SourceCitations = memo(function SourceCitations({ sources }: SourceCitationsProps) {
  if (!sources || sources.length === 0) return null

  return (
    <div className="flex flex-wrap gap-1.5 mt-2">
      {sources.map((source, idx) => (
        <a
          key={idx}
          href={source.url}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 px-2 py-1 bg-background-tertiary hover:bg-accent-blue/10 
                     border border-border-subtle hover:border-accent-blue/30 rounded-full text-xs 
                     text-foreground-secondary hover:text-accent-blue transition-all group"
          title={source.title || source.domain}
        >
          <span className="font-mono text-[10px] opacity-60">
            {source.id}
          </span>
          <span>{source.domain}</span>
          <ExternalLink className="w-3 h-3 opacity-0 group-hover:opacity-100 transition-opacity" />
        </a>
      ))}
    </div>
  )
})