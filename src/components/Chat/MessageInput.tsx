import { useState, useRef, useEffect, KeyboardEvent, ChangeEvent } from 'react';
import { Send, Paperclip, Sparkles, Code, List, Type, Loader2 } from 'lucide-react';

interface MessageInputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  isLoading?: boolean;
  disabled?: boolean;
  placeholder?: string;
  maxTokens?: number;
  currentTokens?: number;
}

type OutputFormat = 'text' | 'code' | 'list';
type CreativityLevel = 'precise' | 'balanced' | 'creative';

export const MessageInput = ({
  value,
  onChange,
  onSubmit,
  isLoading = false,
  disabled = false,
  placeholder = "Type a message...",
  maxTokens = 8000,
  currentTokens = 0
}: MessageInputProps) => {
  const [rows, setRows] = useState(1);
  const [isFocused, setIsFocused] = useState(false);
  const [showFormatOptions, setShowFormatOptions] = useState(false);
  const [outputFormat, setOutputFormat] = useState<OutputFormat>('text');
  const [creativity, setCreativity] = useState<CreativityLevel>('balanced');
  const [showCreativity, setShowCreativity] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  
  // Auto-resize textarea based on content
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      const scrollHeight = textareaRef.current.scrollHeight;
      const lineHeight = 24; // Approximate line height
      const newRows = Math.min(Math.max(Math.floor(scrollHeight / lineHeight), 1), 10);
      setRows(newRows);
      textareaRef.current.style.height = `${scrollHeight}px`;
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
  
  // Calculate token usage percentage
  const tokenPercentage = (currentTokens / maxTokens) * 100;
  const tokenColor = tokenPercentage < 60 ? 'bg-green-500' : 
                     tokenPercentage < 80 ? 'bg-yellow-500' : 'bg-red-500';
  
  // Estimate tokens from input (rough approximation)
  const estimatedInputTokens = Math.ceil(value.length / 4);
  const totalEstimatedTokens = currentTokens + estimatedInputTokens;
  const estimatedPercentage = (totalEstimatedTokens / maxTokens) * 100;
  
  const creativityConfig = {
    precise: { label: 'Precise', temp: 0.2, color: 'text-blue-400' },
    balanced: { label: 'Balanced', temp: 0.6, color: 'text-purple-400' },
    creative: { label: 'Creative', temp: 0.9, color: 'text-pink-400' }
  };
  
  return (
    <div className="relative">
      {/* Floating Action Bar */}
      <div className={`
        absolute -top-12 left-0 right-0 flex items-center justify-between px-4
        transition-all duration-200 ${isFocused ? 'opacity-100' : 'opacity-0 pointer-events-none'}
      `}>
        <div className="flex items-center gap-2">
          {/* Attach File */}
          <button
            className="p-2 hover:bg-white/5 rounded-lg transition-colors"
            title="Attach file (coming soon)"
            disabled
          >
            <Paperclip size={16} className="text-foreground-muted" />
          </button>
          
          {/* Format Options */}
          <div className="relative">
            <button
              onClick={() => setShowFormatOptions(!showFormatOptions)}
              className="p-2 hover:bg-white/5 rounded-lg transition-colors flex items-center gap-1"
              title="Output format"
            >
              {outputFormat === 'text' && <Type size={16} />}
              {outputFormat === 'code' && <Code size={16} />}
              {outputFormat === 'list' && <List size={16} />}
            </button>
            
            {showFormatOptions && (
              <div className="absolute bottom-full mb-2 left-0 bg-background-secondary border border-border-subtle rounded-lg shadow-lg p-1">
                <button
                  onClick={() => { setOutputFormat('text'); setShowFormatOptions(false); }}
                  className="flex items-center gap-2 px-3 py-2 hover:bg-white/5 rounded transition-colors w-full"
                >
                  <Type size={14} /> Text
                </button>
                <button
                  onClick={() => { setOutputFormat('code'); setShowFormatOptions(false); }}
                  className="flex items-center gap-2 px-3 py-2 hover:bg-white/5 rounded transition-colors w-full"
                >
                  <Code size={14} /> Code
                </button>
                <button
                  onClick={() => { setOutputFormat('list'); setShowFormatOptions(false); }}
                  className="flex items-center gap-2 px-3 py-2 hover:bg-white/5 rounded transition-colors w-full"
                >
                  <List size={14} /> List
                </button>
              </div>
            )}
          </div>
          
          {/* Creativity Level */}
          <button
            onClick={() => setShowCreativity(!showCreativity)}
            className={`p-2 hover:bg-white/5 rounded-lg transition-colors flex items-center gap-1 ${creativityConfig[creativity].color}`}
            title="Creativity level"
          >
            <Sparkles size={16} />
            <span className="text-xs">{creativityConfig[creativity].label}</span>
          </button>
          
          {showCreativity && (
            <div className="absolute left-24 bottom-full mb-2 bg-background-secondary border border-border-subtle rounded-lg shadow-lg p-3">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-xs text-foreground-muted">Creativity:</span>
                <span className={`text-xs font-medium ${creativityConfig[creativity].color}`}>
                  {creativityConfig[creativity].label}
                </span>
              </div>
              <div className="flex gap-1">
                <button
                  onClick={() => { setCreativity('precise'); setShowCreativity(false); }}
                  className={`px-3 py-1 rounded text-xs ${creativity === 'precise' ? 'bg-blue-500/20 text-blue-400' : 'hover:bg-white/5'}`}
                >
                  Precise
                </button>
                <button
                  onClick={() => { setCreativity('balanced'); setShowCreativity(false); }}
                  className={`px-3 py-1 rounded text-xs ${creativity === 'balanced' ? 'bg-purple-500/20 text-purple-400' : 'hover:bg-white/5'}`}
                >
                  Balanced
                </button>
                <button
                  onClick={() => { setCreativity('creative'); setShowCreativity(false); }}
                  className={`px-3 py-1 rounded text-xs ${creativity === 'creative' ? 'bg-pink-500/20 text-pink-400' : 'hover:bg-white/5'}`}
                >
                  Creative
                </button>
              </div>
            </div>
          )}
        </div>
        
        {/* Token Counter */}
        <div className={`
          text-xs text-foreground-muted transition-opacity duration-200
          ${value.length > 0 ? 'opacity-100' : 'opacity-0'}
        `}>
          ~{estimatedInputTokens} tokens
        </div>
      </div>
      
      {/* Input Container */}
      <div className={`
        relative bg-background-secondary rounded-2xl border transition-all duration-200
        ${isFocused 
          ? 'border-accent-blue/50 shadow-lg shadow-accent-blue/10 ring-2 ring-accent-blue/20' 
          : 'border-border-subtle/50 hover:border-border-subtle'
        }
      `}>
        {/* Token Usage Bar */}
        {currentTokens > 0 && (
          <div className="absolute top-0 left-4 right-4 h-[2px] bg-background-tertiary rounded-full overflow-hidden">
            <div 
              className={`h-full transition-all duration-500 ${tokenColor}`}
              style={{ width: `${Math.min(estimatedPercentage, 100)}%` }}
            />
          </div>
        )}
        
        {/* Textarea */}
        <textarea
          ref={textareaRef}
          value={value}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          onFocus={() => setIsFocused(true)}
          onBlur={() => setTimeout(() => setIsFocused(false), 200)}
          placeholder={isLoading ? "AI is thinking..." : placeholder}
          disabled={disabled || isLoading}
          rows={rows}
          className={`
            w-full px-5 py-4 pr-14 bg-transparent resize-none
            text-[16px] leading-[24px] placeholder-foreground-muted/60
            focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed
            transition-all duration-200
            ${currentTokens > 0 ? 'pt-6' : ''}
          `}
          style={{
            minHeight: '56px',
            maxHeight: '240px'
          }}
        />
        
        {/* Send Button */}
        <button
          onClick={handleSubmit}
          disabled={disabled || isLoading || !value.trim()}
          className={`
            absolute bottom-3 right-3 p-2.5 rounded-xl
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
        
        {/* Context Warning */}
        {estimatedPercentage > 80 && (
          <div className="absolute -bottom-8 left-0 right-0 flex items-center justify-between px-4">
            <span className="text-xs text-yellow-500">
              Approaching context limit ({Math.round(estimatedPercentage)}% used)
            </span>
            <button className="text-xs text-accent-blue hover:underline">
              Summarize & Continue
            </button>
          </div>
        )}
      </div>
      
      {/* Keyboard Hint */}
      {isFocused && value.length === 0 && (
        <div className="absolute -bottom-6 left-4 text-xs text-foreground-muted/40">
          <kbd className="px-1.5 py-0.5 bg-background-tertiary rounded text-[10px]">Shift</kbd>
          +
          <kbd className="px-1.5 py-0.5 bg-background-tertiary rounded text-[10px]">Enter</kbd>
          for new line
        </div>
      )}
    </div>
  );
};