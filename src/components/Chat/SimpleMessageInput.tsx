import { useState, useRef, useEffect, KeyboardEvent, ChangeEvent, memo } from 'react';
import { Send, Loader2, Square } from 'lucide-react';
import { useStreamingStore } from '../../stores/streaming.store';

interface SimpleMessageInputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  isLoading?: boolean;
  disabled?: boolean;
  placeholder?: string;
}

export const SimpleMessageInput = memo(({
  value,
  onChange,
  onSubmit,
  isLoading = false,
  disabled = false,
  placeholder = "Type a message..."
}: SimpleMessageInputProps) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const { isStreaming, abortStream } = useStreamingStore();
  
  // Auto-resize textarea based on content
  useEffect(() => {
    if (textareaRef.current) {
      // Reset height to get accurate scrollHeight
      textareaRef.current.style.height = '40px';
      const scrollHeight = textareaRef.current.scrollHeight;
      
      // Only grow if content actually needs more space
      if (scrollHeight > 40) {
        const newHeight = Math.min(scrollHeight, 200); // Reduced max height
        textareaRef.current.style.height = `${newHeight}px`;
      }
    }
  }, [value]);
  
  // Focus textarea on mount
  useEffect(() => {
    textareaRef.current?.focus();
  }, []);
  
  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (!disabled && !isLoading && value.trim()) {
        onSubmit();
      }
    }
  };
  
  const handleChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
    onChange(e.target.value);
  };
  
  const handleSubmit = () => {
    if (!disabled && !isLoading && value.trim()) {
      onSubmit();
    }
  };
  
  return (
    <div className="relative">
      <div className="relative bg-background-secondary rounded-xl border border-border-subtle/50 
                    hover:border-border-subtle transition-all duration-200">
        <textarea
          ref={textareaRef}
          value={value}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          placeholder={isLoading ? "AI is thinking..." : placeholder}
          disabled={disabled || isLoading}
          rows={1}
          className="w-full px-3 py-2.5 pr-12 bg-transparent resize-none
                   text-[14px] leading-[22px] placeholder-foreground-muted/60
                   focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed
                   overflow-y-auto"
          style={{
            height: '40px',
            minHeight: '40px',
            maxHeight: '200px'
          }}
        />
        
        {/* Show stop button when streaming, send button otherwise */}
        {isStreaming ? (
          <button
            onClick={() => {
              console.log('[SimpleMessageInput] Force stopping stream and aborting request');
              abortStream(); // This will abort the fetch request and clear the stream
            }}
            className="absolute bottom-2 right-2 p-2 rounded-lg
                     bg-red-500 text-white hover:bg-red-600
                     transition-all duration-200 transform hover:scale-105 active:scale-95"
            title="Stop streaming (Escape)"
          >
            <Square size={18} />
          </button>
        ) : (
          <button
            onClick={handleSubmit}
            disabled={disabled || isLoading || !value.trim()}
            className={`
              absolute bottom-2 right-2 p-2 rounded-lg
              transition-all duration-200 transform
              ${!disabled && !isLoading && value.trim()
                ? 'bg-accent-blue text-white hover:bg-accent-blue/90 hover:scale-105 active:scale-95' 
                : 'bg-background-tertiary text-foreground-muted cursor-not-allowed'
              }
            `}
            title="Send message (Enter)"
          >
            {isLoading ? (
              <Loader2 size={18} className="animate-spin" />
            ) : (
              <Send size={18} />
            )}
          </button>
        )}
      </div>
      
      {/* Simple keyboard hint */}
      {value.length === 0 && (
        <div className="absolute -bottom-5 left-3 text-xs text-foreground-muted/40">
          Press Enter to send, Shift+Enter for new line
        </div>
      )}
    </div>
  );
});