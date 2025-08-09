import { useState, useRef, useEffect, KeyboardEvent, ChangeEvent, memo } from 'react';
import { Send, Loader2, Square, Settings2, Sparkles, Hash } from 'lucide-react';
// Removed store subscriptions to prevent re-renders
// import { useIsStreaming, useStreamActions } from '../../stores/streaming.store';
// import { usePreferencesStore, ResponseStyle } from '../../stores/preferences.store';
import { ConnectionSelector } from '../ConnectionSelector/ConnectionSelector';
import { cn } from '../../lib/utils';
import { useInputDebug } from '../../hooks/useInputDebug';

interface UnifiedComposerProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  isLoading?: boolean;
  disabled?: boolean;
  placeholder?: string;
  connectionId?: string;
  onConnectionChange?: (connectionId: string) => void;
  maxTokens?: number;
}

// Temporarily disabled to fix performance
// const responseStyleOptions: { value: ResponseStyle; label: string; icon: string }[] = [
//   { value: 'ultra-concise', label: 'Ultra Concise', icon: '⚡' },
//   { value: 'concise', label: 'Concise', icon: '📝' },
//   { value: 'balanced', label: 'Balanced', icon: '⚖️' },
//   { value: 'detailed', label: 'Detailed', icon: '📚' },
// ];

export const UnifiedComposer = memo(({
  value,
  onChange,
  onSubmit,
  isLoading = false,
  disabled = false,
  placeholder = "Ask me anything...",
  connectionId,
  onConnectionChange,
  maxTokens = 4096
}: UnifiedComposerProps) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  // Remove store subscriptions to prevent re-renders
  // const isStreaming = useIsStreaming();
  // const { abortStream } = useStreamActions();
  // const { responseStyle, setResponseStyle } = usePreferencesStore();
  const [showSettings, setShowSettings] = useState(false);
  const [localTokens, setLocalTokens] = useState(0);
  
  // Debug input performance
  useInputDebug('UnifiedComposer', value);
  
  // Auto-resize textarea - directly in the change handler for instant feedback
  const adjustHeight = () => {
    if (textareaRef.current) {
      textareaRef.current.style.height = '40px';
      const scrollHeight = textareaRef.current.scrollHeight;
      if (scrollHeight > 40) {
        const newHeight = Math.min(scrollHeight, 200);
        textareaRef.current.style.height = `${newHeight}px`;
      }
    }
  };
  
  // Only calculate tokens when user stops typing for a while
  useEffect(() => {
    // Skip token calculation if actively typing
    if (value.length === 0) {
      setLocalTokens(0);
      return;
    }
    
    const timer = setTimeout(() => {
      const words = value.split(/\s+/).filter(w => w.length > 0).length;
      const estimatedTokens = Math.ceil(words * 1.3);
      setLocalTokens(estimatedTokens);
    }, 1000); // Only calculate after 1 second of no typing
    
    return () => clearTimeout(timer);
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
    const newValue = e.target.value;
    onChange(newValue);
    // Use requestIdleCallback for non-critical UI updates
    if ('requestIdleCallback' in window) {
      requestIdleCallback(adjustHeight);
    } else {
      setTimeout(adjustHeight, 0);
    }
  };
  
  const handleSubmit = () => {
    if (!disabled && !isLoading && value.trim()) {
      onSubmit();
    }
  };
  
  const tokenPercentage = (localTokens / maxTokens) * 100;
  const tokenWarning = tokenPercentage > 80;
  
  return (
    <div className="relative">
      {/* Unified Composer Container */}
      <div 
        className="relative bg-background-secondary rounded-xl border border-border-subtle
                    hover:border-border-subtle/70 transition-all duration-200 shadow-sm"
        role="region"
        aria-label="Message composer"
        id="message-input">
        
        {/* Top Controls Bar */}
        <div className="flex items-center justify-between px-3 py-2 border-b border-border-subtle/30">
          <div className="flex items-center gap-2">
            {/* Connection Selector - Compact Display */}
            {connectionId && onConnectionChange && (
              <div className="text-xs text-foreground-muted flex items-center gap-1.5">
                <Sparkles size={12} className="text-accent-blue/60" />
                <span className="text-foreground-secondary">AI Model</span>
              </div>
            )}
            
            {/* Response Style */}
            <div className="flex items-center gap-1">
              <div className="w-px h-4 bg-border-subtle/50" />
              {/* Response style selector temporarily disabled to fix performance
              <select
                value="balanced"
                onChange={() => {}}
                className="text-xs bg-transparent text-foreground-secondary hover:text-foreground-primary
                         px-2 py-0.5 rounded-md hover:bg-background-tertiary/50 transition-colors
                         focus:outline-none focus:ring-2 focus:ring-accent-blue/50 cursor-pointer"
                title="Response style"
                aria-label="Select response style"
              >
                <option value="balanced">⚖️ Balanced</option>
              </select>
              */}
            </div>
          </div>
          
          {/* Token Counter */}
          <div className="flex items-center gap-2">
            <div className={cn(
              "text-xs flex items-center gap-1",
              tokenWarning ? "text-orange-400" : "text-foreground-muted"
            )}>
              <Hash size={10} />
              <span>{localTokens}</span>
              {tokenWarning && <span className="text-orange-400">!</span>}
            </div>
            
            {/* Settings Toggle */}
            <button
              onClick={() => setShowSettings(!showSettings)}
              className="p-1 hover:bg-background-tertiary/50 rounded-md transition-colors"
              title="Advanced settings"
            >
              <Settings2 size={12} className="text-foreground-muted" />
            </button>
          </div>
        </div>
        
        {/* Advanced Settings Panel */}
        {showSettings && (
          <div className="px-3 py-2 border-b border-border-subtle/30 bg-background-tertiary/30">
            <div className="text-xs text-foreground-muted space-y-2">
              <div className="flex items-center justify-between">
                <span>Max Tokens:</span>
                <input
                  type="number"
                  defaultValue={maxTokens}
                  className="w-20 px-2 py-0.5 bg-background-secondary rounded border border-border-subtle
                           text-foreground-primary text-xs focus:outline-none focus:border-accent-blue/50"
                  min={100}
                  max={32000}
                  readOnly
                />
              </div>
            </div>
          </div>
        )}
        
        {/* Textarea with Integrated Send Button */}
        <div className="relative">
          <textarea
            ref={textareaRef}
            value={value}
            onChange={handleChange}
            onKeyDown={handleKeyDown}
            placeholder={isLoading ? "AI is thinking..." : placeholder}
            disabled={disabled || isLoading}
            rows={1}
            aria-label="Message input"
            aria-describedby="keyboard-hint"
            aria-invalid={false}
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
          
          {/* Token Usage Ring around Send Button */}
          <div className="absolute bottom-2 right-2">
            <div className="relative">
              {/* Token Ring */}
              <svg className="absolute inset-0 w-9 h-9 -rotate-90">
                <circle
                  cx="18"
                  cy="18"
                  r="16"
                  stroke="currentColor"
                  strokeWidth="2"
                  fill="none"
                  className="text-border-subtle/30"
                />
                <circle
                  cx="18"
                  cy="18"
                  r="16"
                  stroke="currentColor"
                  strokeWidth="2"
                  fill="none"
                  strokeDasharray={`${tokenPercentage} ${100 - tokenPercentage}`}
                  className={cn(
                    "transition-all duration-300",
                    tokenWarning ? "text-orange-400" : "text-accent-blue/50"
                  )}
                />
              </svg>
              
              {/* Send/Stop Button */}
              {isLoading ? (
                <button
                  onClick={() => console.log('Abort not connected')}
                  className="relative p-2 rounded-lg bg-red-500 text-white hover:bg-red-600
                           transition-all duration-200 transform hover:scale-105 active:scale-95 button-press"
                  title="Stop streaming (Escape)"
                  aria-label="Stop AI response"
                >
                  <Square size={18} />
                </button>
              ) : (
                <button
                  onClick={handleSubmit}
                  disabled={disabled || isLoading || !value.trim()}
                  className={cn(
                    "relative p-2 rounded-lg transition-all duration-200 transform button-press",
                    !disabled && !isLoading && value.trim()
                      ? 'bg-accent-blue text-white hover:bg-accent-blue/90 hover:scale-105 animate-message-send' 
                      : 'bg-background-tertiary text-foreground-muted cursor-not-allowed'
                  )}
                  title="Send message (Enter)"
                  aria-label="Send message"
                  aria-disabled={disabled || isLoading || !value.trim()}
                >
                  {isLoading ? (
                    <Loader2 size={18} className="animate-spin" />
                  ) : (
                    <Send size={18} className={value.trim() ? 'hover:animate-spring-rotate' : ''} />
                  )}
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
      
      {/* Keyboard hint */}
      {value.length === 0 && (
        <div 
          id="keyboard-hint"
          className="absolute -bottom-5 left-3 text-xs text-foreground-muted/40"
          role="tooltip"
        >
          Press Enter to send, Shift+Enter for new line
        </div>
      )}
    </div>
  );
}, (prevProps, nextProps) => {
  // Custom comparison to prevent unnecessary re-renders
  // Only re-render if these specific props change
  return (
    prevProps.value === nextProps.value &&
    prevProps.isLoading === nextProps.isLoading &&
    prevProps.disabled === nextProps.disabled &&
    prevProps.placeholder === nextProps.placeholder &&
    prevProps.connectionId === nextProps.connectionId
  );
});

UnifiedComposer.displayName = 'UnifiedComposer';