import { memo } from 'react';
import { FileText, AlertCircle, Sparkles } from 'lucide-react';

interface ContextIndicatorProps {
  totalMessages: number;
  includedMessages: number;
  characters: number;
  maxCharacters: number;
  truncated: boolean;
  usingSummary?: boolean;
}

export const ContextIndicator = memo(function ContextIndicator({
  totalMessages,
  includedMessages,
  characters,
  maxCharacters,
  truncated,
  usingSummary
}: ContextIndicatorProps) {
  const usage = Math.round((characters / maxCharacters) * 100);
  const isNearLimit = usage > 80;
  
  if (totalMessages === 0) return null;
  
  return (
    <div className="flex items-center gap-2 text-xs text-foreground-muted">
      <div className="flex items-center gap-1">
        <FileText size={12} />
        <span>
          {includedMessages}/{totalMessages} messages
        </span>
      </div>
      
      {usingSummary && (
        <div className="flex items-center gap-1 text-accent-blue">
          <Sparkles size={12} />
          <span>Using summary</span>
        </div>
      )}
      
      {truncated && !usingSummary && (
        <div className="flex items-center gap-1 text-amber-500">
          <AlertCircle size={12} />
          <span>Context truncated</span>
        </div>
      )}
      
      <div className="flex items-center gap-1">
        <div className="w-16 h-1.5 bg-background-tertiary rounded-full overflow-hidden">
          <div 
            className={`h-full transition-all duration-300 ${
              isNearLimit ? 'bg-amber-500' : 'bg-accent-blue'
            }`}
            style={{ width: `${Math.min(usage, 100)}%` }}
          />
        </div>
        <span className={isNearLimit ? 'text-amber-500' : ''}>
          {usage}%
        </span>
      </div>
    </div>
  );
});