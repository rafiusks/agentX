import { useState, useCallback, memo, useRef } from 'react';
import { UnifiedComposer } from './UnifiedComposer';

interface IsolatedComposerProps {
  onSubmit: (value: string) => void;
  isLoading?: boolean;
  disabled?: boolean;
  placeholder?: string;
  connectionId?: string;
  maxTokens?: number;
}

/**
 * Isolated composer that manages its own input state
 * to prevent re-renders from parent component
 */
export const IsolatedComposer = memo(function IsolatedComposer({
  onSubmit,
  isLoading = false,
  disabled = false,
  placeholder = "Ask me anything...",
  connectionId,
  maxTokens = 4096
}: IsolatedComposerProps) {
  const [localInput, setLocalInput] = useState('');
  const inputRef = useRef(localInput);
  inputRef.current = localInput; // Keep ref in sync
  
  // Stable submit handler that uses ref
  const handleSubmit = useCallback(() => {
    const value = inputRef.current.trim();
    if (value) {
      onSubmit(value);
      setLocalInput(''); // Clear after submit
    }
  }, [onSubmit]);
  
  // Stable change handler
  const handleChange = useCallback((value: string) => {
    setLocalInput(value);
  }, []);
  
  return (
    <UnifiedComposer
      value={localInput}
      onChange={handleChange}
      onSubmit={handleSubmit}
      isLoading={isLoading}
      disabled={disabled}
      placeholder={placeholder}
      connectionId={connectionId}
      maxTokens={maxTokens}
    />
  );
}, (prevProps, nextProps) => {
  // Custom comparison - only re-render when these props change
  return (
    prevProps.isLoading === nextProps.isLoading &&
    prevProps.disabled === nextProps.disabled &&
    prevProps.placeholder === nextProps.placeholder &&
    prevProps.connectionId === nextProps.connectionId &&
    prevProps.onSubmit === nextProps.onSubmit
  );
});