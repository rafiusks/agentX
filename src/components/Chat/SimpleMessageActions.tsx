import { useState } from 'react';
import { Copy, Check } from 'lucide-react';

interface SimpleMessageActionsProps {
  content: string;
}

export const SimpleMessageActions = ({ content }: SimpleMessageActionsProps) => {
  const [copied, setCopied] = useState(false);
  
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('Failed to copy:', error);
    }
  };
  
  return (
    <button
      onClick={handleCopy}
      className="absolute top-2 right-2 p-1.5 opacity-0 group-hover:opacity-100
               hover:bg-white/5 rounded transition-all duration-200"
      title="Copy message"
    >
      {copied ? (
        <Check size={14} className="text-green-400" />
      ) : (
        <Copy size={14} className="text-foreground-muted" />
      )}
    </button>
  );
};