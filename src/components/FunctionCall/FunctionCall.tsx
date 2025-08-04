import React from 'react'
import { ChevronRight } from 'lucide-react'

interface FunctionCallProps {
  name: string
  arguments: string
  isExecuting?: boolean
}

export const FunctionCall: React.FC<FunctionCallProps> = ({ name, arguments: args, isExecuting }) => {
  let parsedArgs: any
  try {
    parsedArgs = JSON.parse(args)
  } catch {
    parsedArgs = args
  }

  return (
    <div className="bg-accent-muted/5 border border-accent/20 rounded-lg p-4 my-2">
      <div className="flex items-center gap-2 mb-2">
        <div className={`w-2 h-2 rounded-full ${isExecuting ? 'bg-accent animate-pulse' : 'bg-accent/50'}`} />
        <span className="text-accent font-mono text-sm">Function Call</span>
        <ChevronRight className="w-4 h-4 text-accent/50" />
        <span className="font-mono text-sm text-foreground">{name}</span>
      </div>
      <div className="ml-4">
        <div className="text-xs text-foreground-secondary mb-1">Arguments:</div>
        <pre className="text-xs font-mono text-foreground-secondary bg-background-secondary rounded p-2 overflow-x-auto">
          {typeof parsedArgs === 'object' ? JSON.stringify(parsedArgs, null, 2) : parsedArgs}
        </pre>
      </div>
    </div>
  )
}