import { memo, useMemo } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { markdownComponents } from './markdownComponents'
import { InlineCitation } from './InlineCitation'

interface MessageWithCitationsProps {
  content: string
  sources: Array<{
    id: string
    domain: string
    url?: string
    title?: string
  }>
}

export const MessageWithCitations = memo(function MessageWithCitations({ 
  content, 
  sources 
}: MessageWithCitationsProps) {
  
  // Process content to add inline citations after quoted content
  const processedContent = useMemo(() => {
    if (!sources || sources.length === 0) return content
    
    let processed = content
    
    // Replace "According to domain:" patterns with citation
    sources.forEach((source) => {
      // Match patterns like "According to domain.com:" or "• According to domain:"
      const patterns = [
        new RegExp(`(According to ${source.domain.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}:)`, 'gi'),
        new RegExp(`(• According to ${source.domain.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}:)`, 'gi'),
      ]
      
      patterns.forEach(pattern => {
        processed = processed.replace(pattern, (match) => {
          return `${match} [${source.id}]`
        })
      })
    })
    
    return processed
  }, [content, sources])
  
  // Custom renderer for text that includes citations
  const renderWithCitations = useMemo(() => {
    return {
      ...markdownComponents,
      p: ({ children }: any) => {
        // Convert children to string to check for citations
        const text = String(children)
        
        // Check if this paragraph contains citation markers
        const citationPattern = /\[(\d+)\]/g
        if (!citationPattern.test(text)) {
          return <p className="mb-4 text-foreground-primary">{children}</p>
        }
        
        // Split text and add citation components
        const parts = text.split(/(\[\d+\])/g)
        const processedChildren = parts.map((part, index) => {
          const citationMatch = part.match(/\[(\d+)\]/)
          if (citationMatch) {
            const sourceId = citationMatch[1]
            const source = sources.find(s => s.id === sourceId)
            if (source) {
              return (
                <InlineCitation
                  key={index}
                  domain={source.domain}
                  url={source.url}
                  number={sourceId}
                />
              )
            }
          }
          return part
        })
        
        return <p className="mb-4 text-foreground-primary">{processedChildren}</p>
      },
      li: ({ children }: any) => {
        // Also handle citations in list items
        const text = String(children)
        const citationPattern = /\[(\d+)\]/g
        
        if (!citationPattern.test(text)) {
          return <li className="pl-2">{children}</li>
        }
        
        const parts = text.split(/(\[\d+\])/g)
        const processedChildren = parts.map((part, index) => {
          const citationMatch = part.match(/\[(\d+)\]/)
          if (citationMatch) {
            const sourceId = citationMatch[1]
            const source = sources.find(s => s.id === sourceId)
            if (source) {
              return (
                <InlineCitation
                  key={index}
                  domain={source.domain}
                  url={source.url}
                  number={sourceId}
                />
              )
            }
          }
          return part
        })
        
        return <li className="pl-2">{processedChildren}</li>
      }
    }
  }, [sources])
  
  return (
    <div className="prose-chat" data-message-content>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={renderWithCitations}
      >
        {processedContent}
      </ReactMarkdown>
    </div>
  )
})