import React from 'react';
import { FileText, Code, Database, Settings, GitBranch } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useCanvasStore } from '@/stores/canvas.store';

export const CanvasToolbar: React.FC = () => {
  const {
    canvasType,
    canvasMode,
    showLineNumbers,
    wordWrap,
    fontSize,
    theme,
    currentArtifact,
    setCanvasType,
    setCanvasMode,
    toggleLineNumbers,
    toggleWordWrap,
    setFontSize,
    setTheme,
  } = useCanvasStore();


  return (
    <div className="flex items-center gap-2 px-4 py-2 border-b border-border-subtle bg-background-primary">
      {/* Canvas type selector */}
      <div className="flex items-center gap-1 p-1 bg-background-secondary rounded-lg">
        <Button
          variant={canvasType === 'code' ? 'default' : 'ghost'}
          size="sm"
          onClick={() => setCanvasType('code')}
          className="h-7 px-2"
        >
          <Code size={14} className="mr-1" />
          Code
        </Button>
        
        <Button
          variant={canvasType === 'document' ? 'default' : 'ghost'}
          size="sm"
          onClick={() => setCanvasType('document')}
          className="h-7 px-2"
        >
          <FileText size={14} className="mr-1" />
          Document
        </Button>
        
        <Button
          variant={canvasType === 'data' ? 'default' : 'ghost'}
          size="sm"
          onClick={() => setCanvasType('data')}
          className="h-7 px-2"
        >
          <Database size={14} className="mr-1" />
          Data
        </Button>
      </div>

      <div className="w-px h-4 bg-border-subtle" />

      {/* View mode */}
      <div className="flex items-center gap-1">
        <Button
          variant={canvasMode === 'edit' ? 'default' : 'ghost'}
          size="sm"
          onClick={() => setCanvasMode('edit')}
          className="h-7 px-2"
        >
          Edit
        </Button>
        
        <Button
          variant={canvasMode === 'view' ? 'default' : 'ghost'}
          size="sm"
          onClick={() => setCanvasMode('view')}
          className="h-7 px-2"
        >
          View
        </Button>
        
        <Button
          variant={canvasMode === 'diff' ? 'default' : 'ghost'}
          size="sm"
          onClick={() => setCanvasMode('diff')}
          disabled={!currentArtifact || currentArtifact.version === 1}
          className="h-7 px-2"
        >
          <GitBranch size={14} className="mr-1" />
          Diff
        </Button>
      </div>

      <div className="flex-1" />

      {/* Settings dropdown */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm">
            <Settings size={16} />
          </Button>
        </DropdownMenuTrigger>
        
        <DropdownMenuContent className="w-64">
          <DropdownMenuLabel>Canvas Settings</DropdownMenuLabel>
          <DropdownMenuSeparator />
          
          {/* Font size */}
          <div className="px-2 py-2">
            <Label className="text-xs">Font Size: {fontSize}px</Label>
            <input
              type="range"
              value={fontSize}
              onChange={(e) => setFontSize(Number(e.target.value))}
              min={10}
              max={24}
              step={1}
              className="mt-2 w-full"
            />
          </div>
          
          <DropdownMenuSeparator />
          
          {/* Line numbers (code only) */}
          {canvasType === 'code' && (
            <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
              <div className="flex items-center justify-between w-full">
                <Label htmlFor="line-numbers" className="text-sm">Line Numbers</Label>
                <Switch
                  id="line-numbers"
                  checked={showLineNumbers}
                  onCheckedChange={toggleLineNumbers}
                />
              </div>
            </DropdownMenuItem>
          )}
          
          {/* Word wrap */}
          <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
            <div className="flex items-center justify-between w-full">
              <Label htmlFor="word-wrap" className="text-sm">Word Wrap</Label>
              <Switch
                id="word-wrap"
                checked={wordWrap}
                onCheckedChange={toggleWordWrap}
              />
            </div>
          </DropdownMenuItem>
          
          <DropdownMenuSeparator />
          
          {/* Theme */}
          <DropdownMenuItem onClick={() => setTheme('light')}>
            <span className={theme === 'light' ? 'font-bold' : ''}>Light Theme</span>
          </DropdownMenuItem>
          
          <DropdownMenuItem onClick={() => setTheme('dark')}>
            <span className={theme === 'dark' ? 'font-bold' : ''}>Dark Theme</span>
          </DropdownMenuItem>
          
          <DropdownMenuItem onClick={() => setTheme('auto')}>
            <span className={theme === 'auto' ? 'font-bold' : ''}>Auto Theme</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
};