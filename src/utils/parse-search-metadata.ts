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

interface ParsedMessage {
  content: string
  searchMetadata?: SearchMetadata
  sources: Array<{
    id: string
    domain: string
    url?: string
    title?: string
  }>
}

export function parseSearchMetadata(content: string): ParsedMessage {
  const lines = content.split('\n')
  const searchMetadata: Partial<SearchMetadata> = {}
  const results: SearchResult[] = []
  const sources: ParsedMessage['sources'] = []
  let cleanContent = content
  let inSearchResults = false
  let currentResult: Partial<SearchResult> | null = null
  
  // Check if this contains search results
  const searchHeaderIndex = lines.findIndex(line => 
    line.includes('WEB SEARCH RESULTS') || 
    line.includes('Search Results') ||
    line.includes('Found') && line.includes('results')
  )
  
  if (searchHeaderIndex === -1) {
    // No search results, but check for inline citations
    const citationPattern = /\(Source:\s*RESULT\s*(\d+),\s*([^)]+)\)/g
    let match
    const seenSources = new Set<string>()
    
    while ((match = citationPattern.exec(content)) !== null) {
      const sourceKey = `${match[1]}-${match[2]}`
      if (!seenSources.has(sourceKey)) {
        seenSources.add(sourceKey)
        sources.push({
          id: `R${match[1]}`,
          domain: match[2].trim(),
        })
      }
    }
    
    return { content, sources }
  }
  
  // Parse search results section
  let searchEndIndex = lines.length
  
  for (let i = searchHeaderIndex; i < lines.length; i++) {
    const line = lines[i].trim()
    
    // Check for provider info
    if (line.includes('Provider:') && line.includes('|')) {
      const parts = line.split('|')
      const providerPart = parts[0].replace('Provider:', '').trim()
      searchMetadata.provider = providerPart
      
      const resultsPart = parts[1]?.match(/Results:\s*(\d+)/)
      if (resultsPart) {
        searchMetadata.resultCount = parseInt(resultsPart[1])
      }
      inSearchResults = true
    }
    
    // Check for DuckDuckGo format
    else if (line.includes('from DuckDuckGo')) {
      searchMetadata.provider = 'DuckDuckGo'
      inSearchResults = true
    }
    
    // Check for result count
    else if (line.match(/Found\s+(\d+)\s+results/)) {
      const match = line.match(/Found\s+(\d+)\s+results/)
      if (match) {
        searchMetadata.resultCount = parseInt(match[1])
      }
    }
    
    // Parse individual results
    else if (line.match(/^\[?RESULT\s*(\d+)\]?/) || line.match(/^📌\s*RESULT\s*(\d+)/)) {
      // Save previous result if exists
      if (currentResult && currentResult.title) {
        results.push(currentResult as SearchResult)
      }
      currentResult = {}
    }
    
    // Parse result fields
    else if (currentResult && inSearchResults) {
      if (line.startsWith('Title:')) {
        currentResult.title = line.replace('Title:', '').trim()
      } else if (line.startsWith('Domain:') || line.startsWith('Source:')) {
        currentResult.domain = line.replace(/^(Domain|Source):/, '').trim()
      } else if (line.startsWith('URL:')) {
        currentResult.url = line.replace('URL:', '').trim()
      } else if (line.startsWith('Summary:')) {
        currentResult.summary = line.replace('Summary:', '').trim()
      }
    }
    
    // Check for end of search results
    if (line.includes('Based on these search results') || 
        line.includes('provide a comprehensive answer')) {
      searchEndIndex = i
      // Save last result if exists
      if (currentResult && currentResult.title) {
        results.push(currentResult as SearchResult)
      }
      break
    }
  }
  
  // Save last result if we reached the end
  if (currentResult && currentResult.title && searchEndIndex === lines.length) {
    results.push(currentResult as SearchResult)
  }
  
  // Remove search results section from content
  if (inSearchResults) {
    const contentLines = [
      ...lines.slice(0, searchHeaderIndex),
      ...lines.slice(searchEndIndex)
    ].filter(line => !line.includes('Based on these search results'))
    
    cleanContent = contentLines.join('\n').trim()
  }
  
  // Extract inline citations from the clean content
  const citationPattern = /\(Source:\s*RESULT\s*(\d+),\s*([^)]+)\)/g
  let match
  const seenSources = new Set<string>()
  
  while ((match = citationPattern.exec(cleanContent)) !== null) {
    const resultNum = parseInt(match[1]) - 1
    const domain = match[2].trim()
    const sourceKey = `${match[1]}-${domain}`
    
    if (!seenSources.has(sourceKey)) {
      seenSources.add(sourceKey)
      const result = results[resultNum]
      sources.push({
        id: `R${match[1]}`,
        domain: domain,
        url: result?.url,
        title: result?.title
      })
    }
  }
  
  // Build final search metadata if we found results
  const finalMetadata = results.length > 0 ? {
    provider: searchMetadata.provider || 'Web Search',
    resultCount: searchMetadata.resultCount || results.length,
    results
  } : undefined
  
  return {
    content: cleanContent,
    searchMetadata: finalMetadata,
    sources
  }
}