import React, { useEffect, useRef } from 'react';
import { X, Save, Download, Copy, Maximize2, Minimize2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useCanvasStore } from '@/stores/canvas.store';
import { CodeCanvas } from './CodeCanvas';
import { DocumentCanvas } from './DocumentCanvas';
import { DataCanvas } from './DataCanvas';
import { CanvasToolbar } from './CanvasToolbar';
import { VersionHistory } from './VersionHistory';
import { cn } from '@/lib/utils';

export const Canvas: React.FC = () => {
  const {
    isCanvasOpen,
    canvasWidth,
    currentArtifact,
    canvasType,
    canvasMode,
    closeCanvas,
    setCanvasWidth,
    saveArtifact,
  } = useCanvasStore();

  const resizerRef = useRef<HTMLDivElement>(null);
  const [isResizing, setIsResizing] = React.useState(false);
  const [isFullscreen, setIsFullscreen] = React.useState(false);

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      
      const screenWidth = window.innerWidth;
      const newWidth = ((screenWidth - e.clientX) / screenWidth) * 100;
      setCanvasWidth(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
    };

    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isResizing, setCanvasWidth]);

  const handleExport = () => {
    if (!currentArtifact) return;
    
    const blob = new Blob([currentArtifact.content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${currentArtifact.title.replace(/\s+/g, '_')}.${getFileExtension()}`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleCopy = async () => {
    if (!currentArtifact) return;
    
    try {
      await navigator.clipboard.writeText(currentArtifact.content);
      // TODO: Show toast notification
    } catch (error) {
      console.error('Failed to copy:', error);
    }
  };

  const getFileExtension = () => {
    if (!currentArtifact) return 'txt';
    
    switch (currentArtifact.type) {
      case 'code':
        return currentArtifact.language || 'txt';
      case 'document':
        return 'md';
      case 'data':
        return 'json';
      default:
        return 'txt';
    }
  };

  const toggleFullscreen = () => {
    if (isFullscreen) {
      setCanvasWidth(50);
    } else {
      setCanvasWidth(100);
    }
    setIsFullscreen(!isFullscreen);
  };

  if (!isCanvasOpen) return null;

  const renderCanvas = () => {
    switch (canvasType) {
      case 'code':
        return <CodeCanvas />;
      case 'document':
        return <DocumentCanvas />;
      case 'data':
        return <DataCanvas />;
      default:
        return <DocumentCanvas />;
    }
  };

  return (
    <div 
      className={cn(
        "fixed right-0 top-0 h-full bg-background-primary border-l border-border-subtle flex flex-col z-40 transition-all duration-300",
        isFullscreen ? "w-full" : ""
      )}
      style={{ width: isFullscreen ? '100%' : `${canvasWidth}%` }}
    >
      {/* Resize handle */}
      {!isFullscreen && (
        <div
          ref={resizerRef}
          className="absolute left-0 top-0 h-full w-1 cursor-ew-resize hover:bg-accent-blue/50 transition-colors"
          onMouseDown={() => setIsResizing(true)}
        />
      )}

      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-border-subtle bg-background-secondary">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium text-foreground-primary">
            {currentArtifact?.title || 'Canvas'}
          </h3>
          {currentArtifact && (
            <span className="text-xs text-foreground-tertiary">
              v{currentArtifact.version} • {currentArtifact.type}
            </span>
          )}
        </div>

        <div className="flex items-center gap-1">
          {currentArtifact?.isDirty && (
            <span className="text-xs text-accent-orange px-2">Unsaved changes</span>
          )}
          
          <Button
            variant="ghost"
            size="sm"
            onClick={saveArtifact}
            disabled={!currentArtifact?.isDirty}
            title="Save"
          >
            <Save size={16} />
          </Button>

          <Button
            variant="ghost"
            size="sm"
            onClick={handleCopy}
            disabled={!currentArtifact}
            title="Copy to clipboard"
          >
            <Copy size={16} />
          </Button>

          <Button
            variant="ghost"
            size="sm"
            onClick={handleExport}
            disabled={!currentArtifact}
            title="Export"
          >
            <Download size={16} />
          </Button>

          <Button
            variant="ghost"
            size="sm"
            onClick={toggleFullscreen}
            title={isFullscreen ? "Exit fullscreen" : "Fullscreen"}
          >
            {isFullscreen ? <Minimize2 size={16} /> : <Maximize2 size={16} />}
          </Button>

          <Button
            variant="ghost"
            size="sm"
            onClick={closeCanvas}
            title="Close canvas"
          >
            <X size={16} />
          </Button>
        </div>
      </div>

      {/* Toolbar */}
      <CanvasToolbar />

      {/* Main canvas area */}
      <div className="flex-1 flex overflow-hidden">
        <div className="flex-1 overflow-auto">
          {currentArtifact ? (
            renderCanvas()
          ) : (
            <div className="flex items-center justify-center h-full">
              <div className="text-center space-y-4 max-w-md">
                <div className="text-6xl">📝</div>
                <h2 className="text-xl font-semibold text-foreground-primary">
                  No artifact loaded
                </h2>
                <p className="text-foreground-secondary">
                  Create or load an artifact to start editing
                </p>
                <div className="flex gap-2 justify-center">
                  <Button
                    onClick={() => useCanvasStore.getState().createArtifact('code', 'Untitled', '', 'javascript')}
                  >
                    New Code File
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => useCanvasStore.getState().createArtifact('document', 'Untitled', '')}
                  >
                    New Document
                  </Button>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Version history sidebar */}
        {canvasMode === 'diff' && <VersionHistory />}
      </div>
    </div>
  );
};