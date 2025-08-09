import { useState } from 'react';
import { Copy, Check, RefreshCw } from 'lucide-react';

interface SimpleMessageActionsProps {
  content: string;
  isAssistant?: boolean;
  messageId?: string;
  onRegenerate?: () => void;
}

export const SimpleMessageActions = ({ 
  content, 
  isAssistant = false,
  messageId,
  onRegenerate 
}: SimpleMessageActionsProps) => {
  const [copied, setCopied] = useState(false);
  const [isRegenerating, setIsRegenerating] = useState(false);
  
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('Failed to copy:', error);
    }
  };
  
  const handleRegenerate = () => {
    if (onRegenerate) {
      setIsRegenerating(true);
      onRegenerate();
      // Reset after animation
      setTimeout(() => setIsRegenerating(false), 1000);
    }
  };
  
  return (
    <div className="absolute top-2 right-2 flex items-center gap-1 opacity-0 group-hover:opacity-100
                    transition-all duration-200">
      {/* Regenerate button for assistant messages */}
      {isAssistant && onRegenerate && (
        <button
          onClick={handleRegenerate}
          className="p-1.5 hover:bg-white/5 rounded-xl transition-all duration-200 button-press"
          title="Regenerate response"
          disabled={isRegenerating}
          data-regenerate-button
        >
          <RefreshCw 
            size={14} 
            className={`text-foreground-muted hover:text-foreground-secondary 
                      ${isRegenerating ? 'animate-spin' : 'hover:animate-spring-rotate'}`} 
          />
        </button>
      )}
      
      {/* Copy button */}
      <button
        onClick={handleCopy}
        className="p-1.5 hover:bg-white/5 rounded-xl transition-all duration-200 button-press"
        title="Copy message"
        data-copy-button
      >
        {copied ? (
          <Check size={14} className="text-green-400 animate-copy-confirm" />
        ) : (
          <Copy size={14} className="text-foreground-muted hover:text-foreground-secondary" />
        )}
      </button>
    </div>
  );
};