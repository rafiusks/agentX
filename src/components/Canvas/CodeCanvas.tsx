import React, { useRef, useEffect, useCallback } from 'react';
import { useCanvasStore } from '@/stores/canvas.store';
import { cn } from '@/lib/utils';

// Simple code editor with syntax highlighting using Monaco-like styling
export const CodeCanvas: React.FC = () => {
  const {
    currentArtifact,
    updateArtifact,
    showLineNumbers,
    wordWrap,
    fontSize,
  } = useCanvasStore();

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const lineNumbersRef = useRef<HTMLPreElement>(null);

  const updateLineNumbers = useCallback(() => {
    if (!lineNumbersRef.current || !currentArtifact) return;
    
    const lines = currentArtifact.content.split('\n').length;
    const lineNumbers = Array.from({ length: lines }, (_, i) => i + 1).join('\n');
    lineNumbersRef.current.textContent = lineNumbers;
  }, [currentArtifact]);

  useEffect(() => {
    updateLineNumbers();
  }, [updateLineNumbers]);

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    updateArtifact(e.target.value);
    updateLineNumbers();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Handle tab key
    if (e.key === 'Tab') {
      e.preventDefault();
      const start = e.currentTarget.selectionStart;
      const end = e.currentTarget.selectionEnd;
      const value = e.currentTarget.value;
      
      // Insert tab character
      const newValue = value.substring(0, start) + '  ' + value.substring(end);
      updateArtifact(newValue);
      
      // Move cursor
      setTimeout(() => {
        if (textareaRef.current) {
          textareaRef.current.selectionStart = textareaRef.current.selectionEnd = start + 2;
        }
      }, 0);
    }
    
    // Handle Cmd/Ctrl+S for save
    if ((e.metaKey || e.ctrlKey) && e.key === 's') {
      e.preventDefault();
      useCanvasStore.getState().saveArtifact();
    }
  };

  const getLanguageClass = () => {
    const language = currentArtifact?.language || 'plaintext';
    return `language-${language}`;
  };

  if (!currentArtifact) return null;

  return (
    <div className="flex h-full bg-background-primary">
      {showLineNumbers && (
        <div className="flex-shrink-0 bg-background-secondary border-r border-border-subtle">
          <pre
            ref={lineNumbersRef}
            className="px-3 py-4 text-foreground-tertiary select-none"
            style={{ 
              fontSize: `${fontSize}px`,
              lineHeight: '1.5',
              fontFamily: 'JetBrains Mono, Monaco, Consolas, monospace',
            }}
          />
        </div>
      )}
      
      <div className="flex-1 relative">
        <textarea
          ref={textareaRef}
          value={currentArtifact.content}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          className={cn(
            "w-full h-full px-4 py-4 bg-transparent resize-none outline-none",
            "text-foreground-primary placeholder-foreground-tertiary",
            getLanguageClass(),
            wordWrap && "whitespace-pre-wrap break-words"
          )}
          style={{ 
            fontSize: `${fontSize}px`,
            lineHeight: '1.5',
            fontFamily: 'JetBrains Mono, Monaco, Consolas, monospace',
            tabSize: 2,
          }}
          placeholder="Start typing your code here..."
          spellCheck={false}
        />
        
        {/* Language indicator */}
        <div className="absolute bottom-2 right-2 px-2 py-1 bg-background-secondary rounded text-xs text-foreground-secondary">
          {currentArtifact.language || 'plaintext'}
        </div>
      </div>
    </div>
  );
};