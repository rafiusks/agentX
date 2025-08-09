import { useState, useCallback, memo, useRef, useEffect } from 'react';
import { Send, Square } from 'lucide-react';
import { useIsStreaming, useStreamActions } from '../../stores/streaming.store';
import { usePreferencesStore } from '../../stores/preferences.store';

interface ComposerWithFeaturesProps {
  onSubmit: (value: string) => void;
  isLoading?: boolean;
  disabled?: boolean;
  placeholder?: string;
  connectionId?: string;
  maxTokens?: number;
}

/**
 * Split the composer into two parts:
 * 1. Input component - handles text input only (no re-renders from stores)
 * 2. Controls component - handles streaming state and preferences (can re-render)
 */

// Pure input component that doesn't subscribe to any stores
const PureInput = memo(function PureInput({
  value,
  onChange,
  onSubmit,
  onKeyDown,
  placeholder,
  disabled
}: {
  value: string;
  onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void;
  onSubmit: () => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLTextAreaElement>) => void;
  placeholder?: string;
  disabled?: boolean;
}) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  
  // Auto-resize directly in the DOM without React re-render
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
  
  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    onChange(e);
    requestAnimationFrame(adjustHeight);
  };
  
  return (
    <textarea
      ref={textareaRef}
      value={value}
      onChange={handleChange}
      onKeyDown={onKeyDown}
      placeholder={placeholder}
      disabled={disabled}
      rows={1}
      aria-label="Message input"
      className="w-full px-3 py-2.5 pr-12 bg-transparent resize-none
                 text-[14px] leading-[22px] placeholder-foreground-muted/60
                 outline-none disabled:opacity-50 disabled:cursor-not-allowed
                 overflow-y-auto"
      style={{ height: '40px', minHeight: '40px', maxHeight: '200px' }}
    />
  );
});

// Controls that can re-render without affecting input performance
const ComposerControls = memo(function ComposerControls({
  onSubmit,
  canSubmit,
  connectionId,
  tokenCount
}: {
  onSubmit: () => void;
  canSubmit: boolean;
  connectionId?: string;
  tokenCount?: number;
}) {
  const isStreaming = useIsStreaming();
  const { abortStream } = useStreamActions();
  const { responseStyle, setResponseStyle } = usePreferencesStore();
  
  return (
    <div className="absolute bottom-2 right-2">
      {isStreaming ? (
        <button
          onClick={() => abortStream()}
          className="relative p-2 rounded-lg bg-red-500 text-white hover:bg-red-600
                   transition-all duration-200"
          title="Stop streaming"
        >
          <Square size={16} />
        </button>
      ) : (
        <button
          onClick={onSubmit}
          disabled={!canSubmit}
          className={`relative p-2 rounded-lg transition-all duration-200
                    ${canSubmit 
                      ? 'bg-accent-blue text-white hover:bg-accent-blue/90' 
                      : 'bg-background-tertiary text-foreground-muted cursor-not-allowed'}`}
          title="Send message"
        >
          <Send size={16} />
        </button>
      )}
    </div>
  );
});

export const ComposerWithFeatures = memo(function ComposerWithFeatures({
  onSubmit,
  isLoading = false,
  disabled = false,
  placeholder = "Ask me anything...",
  connectionId,
  maxTokens = 4096
}: ComposerWithFeaturesProps) {
  const [localInput, setLocalInput] = useState('');
  const [tokenCount, setTokenCount] = useState(0);
  const inputRef = useRef(localInput);
  inputRef.current = localInput;
  
  // Debounced token calculation
  useEffect(() => {
    const timer = setTimeout(() => {
      const words = localInput.split(/\s+/).filter(w => w.length > 0).length;
      setTokenCount(Math.ceil(words * 1.3));
    }, 1000);
    return () => clearTimeout(timer);
  }, [localInput]);
  
  const handleSubmit = useCallback(() => {
    const value = inputRef.current.trim();
    if (value && !isLoading && !disabled) {
      onSubmit(value);
      setLocalInput('');
      setTokenCount(0);
    }
  }, [onSubmit, isLoading, disabled]);
  
  const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  }, [handleSubmit]);
  
  const handleChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setLocalInput(e.target.value);
  }, []);
  
  const canSubmit = localInput.trim().length > 0 && !isLoading && !disabled;
  
  return (
    <div className="relative bg-background-secondary rounded-xl border border-border-subtle
                    hover:border-border-subtle/70 transition-all duration-200 shadow-sm"
         role="region" aria-label="Message composer">
      {/* Header with response style and token counter */}
      <ResponseStyleHeader tokenCount={tokenCount} maxTokens={maxTokens} />
      
      {/* Input area */}
      <div className="relative">
        <PureInput
          value={localInput}
          onChange={handleChange}
          onSubmit={handleSubmit}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled || isLoading}
        />
        <ComposerControls
          onSubmit={handleSubmit}
          canSubmit={canSubmit}
          connectionId={connectionId}
          tokenCount={tokenCount}
        />
      </div>
    </div>
  );
});

// Separate header component that can re-render independently
const ResponseStyleHeader = memo(function ResponseStyleHeader({
  tokenCount,
  maxTokens
}: {
  tokenCount: number;
  maxTokens: number;
}) {
  const { responseStyle, setResponseStyle } = usePreferencesStore();
  const tokenPercentage = (tokenCount / maxTokens) * 100;
  const tokenWarning = tokenPercentage > 80;
  
  return (
    <div className="flex items-center justify-between px-3 py-2 border-b border-border-subtle/30">
      <div className="flex items-center gap-2">
        <select
          value={responseStyle}
          onChange={(e) => setResponseStyle(e.target.value as any)}
          className="text-xs bg-transparent text-foreground-secondary hover:text-foreground-primary
                   px-2 py-0.5 rounded-md hover:bg-background-tertiary/50 transition-colors
                   outline-none cursor-pointer"
          title="Response style"
        >
          <option value="ultra-concise">⚡ Ultra Concise</option>
          <option value="concise">📝 Concise</option>
          <option value="balanced">⚖️ Balanced</option>
          <option value="detailed">📚 Detailed</option>
        </select>
      </div>
      
      <div className={`text-xs flex items-center gap-1 ${tokenWarning ? 'text-orange-400' : 'text-foreground-muted'}`}>
        <span>{tokenCount} / {maxTokens}</span>
        {tokenWarning && <span>⚠️</span>}
      </div>
    </div>
  );
});