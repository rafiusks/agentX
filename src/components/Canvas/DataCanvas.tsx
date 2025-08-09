import React, { useState, useEffect, useCallback } from 'react';
import { AlertCircle, Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useCanvasStore } from '@/stores/canvas.store';
import { cn } from '@/lib/utils';

export const DataCanvas: React.FC = () => {
  const {
    currentArtifact,
    updateArtifact,
    fontSize,
  } = useCanvasStore();

  const [isValid, setIsValid] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [_formatted, setFormatted] = useState('');

  const validateAndFormat = useCallback(() => {
    if (!currentArtifact) return;
    
    try {
      const parsed = JSON.parse(currentArtifact.content || '{}');
      setFormatted(JSON.stringify(parsed, null, 2));
      setIsValid(true);
      setError(null);
    } catch (e) {
      setIsValid(false);
      setError(e instanceof Error ? e.message : 'Invalid JSON');
      setFormatted(currentArtifact.content);
    }
  }, [currentArtifact]);

  useEffect(() => {
    if (!currentArtifact) return;
    validateAndFormat();
  }, [currentArtifact, validateAndFormat]);

  const handleFormat = () => {
    if (!currentArtifact) return;
    
    try {
      const parsed = JSON.parse(currentArtifact.content);
      const formatted = JSON.stringify(parsed, null, 2);
      updateArtifact(formatted);
    } catch (e) {
      // Already handled by validation
    }
  };

  const handleMinify = () => {
    if (!currentArtifact) return;
    
    try {
      const parsed = JSON.parse(currentArtifact.content);
      const minified = JSON.stringify(parsed);
      updateArtifact(minified);
    } catch (e) {
      // Already handled by validation
    }
  };

  const handleCopyPath = (path: string) => {
    navigator.clipboard.writeText(path);
  };

  const renderJsonTree = (data: unknown, path = '$', depth = 0): React.ReactNode => {
    if (data === null) return <span className="text-foreground-tertiary">null</span>;
    if (data === undefined) return <span className="text-foreground-tertiary">undefined</span>;
    
    if (typeof data === 'object') {
      const isArray = Array.isArray(data);
      const entries = Object.entries(data);
      
      if (entries.length === 0) {
        return <span className="text-foreground-tertiary">{isArray ? '[]' : '{}'}</span>;
      }
      
      return (
        <div className="ml-4">
          {entries.map(([key, value], index) => (
            <div key={index} className="group hover:bg-background-secondary/50 rounded px-1">
              <span 
                className="text-foreground-secondary cursor-pointer"
                onClick={() => handleCopyPath(`${path}${isArray ? `[${key}]` : `.${key}`}`)}
                title="Click to copy path"
              >
                {isArray ? `[${key}]` : `"${key}"`}:
              </span>
              {' '}
              {typeof value === 'object' && value !== null ? (
                <span className="text-foreground-tertiary">
                  {Array.isArray(value) ? `Array[${value.length}]` : 'Object'}
                </span>
              ) : (
                renderJsonTree(value, `${path}${isArray ? `[${key}]` : `.${key}`}`, depth + 1)
              )}
              {typeof value === 'object' && value !== null && (
                renderJsonTree(value, `${path}${isArray ? `[${key}]` : `.${key}`}`, depth + 1)
              )}
            </div>
          ))}
        </div>
      );
    }
    
    if (typeof data === 'string') {
      return <span className="text-green-500">"{data}"</span>;
    }
    
    if (typeof data === 'number') {
      return <span className="text-blue-500">{data}</span>;
    }
    
    if (typeof data === 'boolean') {
      return <span className="text-purple-500">{data.toString()}</span>;
    }
    
    return <span>{String(data)}</span>;
  };

  if (!currentArtifact) return null;

  return (
    <div className="flex flex-col h-full bg-background-primary">
      {/* Toolbar */}
      <div className="flex items-center gap-2 px-4 py-2 border-b border-border-subtle bg-background-secondary">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleFormat}
          disabled={!isValid}
          title="Format JSON"
        >
          Format
        </Button>
        
        <Button
          variant="ghost"
          size="sm"
          onClick={handleMinify}
          disabled={!isValid}
          title="Minify JSON"
        >
          Minify
        </Button>
        
        <div className="flex-1" />
        
        {isValid ? (
          <div className="flex items-center gap-1 text-xs text-accent-green">
            <Check size={14} />
            <span>Valid JSON</span>
          </div>
        ) : (
          <div className="flex items-center gap-1 text-xs text-accent-red">
            <AlertCircle size={14} />
            <span>Invalid JSON</span>
          </div>
        )}
      </div>

      {/* Error message */}
      {error && (
        <div className="px-4 py-2 bg-accent-red/10 border-b border-accent-red/20">
          <p className="text-sm text-accent-red">{error}</p>
        </div>
      )}

      {/* Content area */}
      <div className="flex-1 flex overflow-hidden">
        {/* Editor */}
        <div className="flex-1 overflow-auto">
          <textarea
            value={currentArtifact.content}
            onChange={(e) => updateArtifact(e.target.value)}
            className={cn(
              "w-full h-full px-4 py-4 bg-transparent resize-none outline-none",
              "text-foreground-primary placeholder-foreground-tertiary",
              "font-mono",
              !isValid && "text-accent-red"
            )}
            style={{ 
              fontSize: `${fontSize}px`,
              lineHeight: '1.5',
              fontFamily: 'JetBrains Mono, Monaco, Consolas, monospace',
              tabSize: 2,
            }}
            placeholder='{\n  "key": "value"\n}'
            spellCheck={false}
          />
        </div>

        {/* Tree view */}
        {isValid && (
          <div className="w-1/3 border-l border-border-subtle overflow-auto">
            <div className="p-4">
              <h3 className="text-sm font-medium text-foreground-primary mb-2">Tree View</h3>
              <div className="text-sm">
                {(() => {
                  try {
                    const parsed = JSON.parse(currentArtifact.content || '{}');
                    return renderJsonTree(parsed);
                  } catch {
                    return null;
                  }
                })()}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};