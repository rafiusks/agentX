import { memo } from 'react'
import { ExternalLink, Globe } from 'lucide-react'

interface InlineCitationProps {
  domain: string
  url?: string
  number?: string | number
}

export const InlineCitation = memo(function InlineCitation({ 
  domain, 
  url, 
  number 
}: InlineCitationProps) {
  const content = (
    <span className="inline-flex items-center gap-0.5 px-1.5 py-0.5 ml-1 
                     bg-accent-blue/10 border border-accent-blue/20 
                     rounded text-[10px] font-medium text-accent-blue
                     hover:bg-accent-blue/20 hover:border-accent-blue/30 
                     transition-all group cursor-pointer align-baseline">
      <Globe className="w-2.5 h-2.5" />
      {number && <span className="font-mono opacity-70">{number}</span>}
      <span className="max-w-[100px] truncate">{domain}</span>
      {url && <ExternalLink className="w-2.5 h-2.5 opacity-0 group-hover:opacity-100 transition-opacity" />}
    </span>
  )
  
  if (url) {
    return (
      <a 
        href={url} 
        target="_blank" 
        rel="noopener noreferrer"
        className="inline-block"
        title={`Source: ${domain}`}
      >
        {content}
      </a>
    )
  }
  
  return content
})