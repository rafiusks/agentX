import React, { useState, useRef } from 'react';
import ReactMarkdown from 'react-markdown';
import { Bold, Italic, List, ListOrdered, Quote, Code, Link, Image, Eye, Edit3 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useCanvasStore } from '@/stores/canvas.store';
import { cn } from '@/lib/utils';

export const DocumentCanvas: React.FC = () => {
  const {
    currentArtifact,
    updateArtifact,
    fontSize,
    wordWrap,
  } = useCanvasStore();

  const [isPreview, setIsPreview] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const insertMarkdown = (before: string, after = '') => {
    if (!textareaRef.current) return;
    
    const textarea = textareaRef.current;
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const selectedText = textarea.value.substring(start, end);
    const value = textarea.value;
    
    const newValue = 
      value.substring(0, start) + 
      before + 
      (selectedText || 'text') + 
      after + 
      value.substring(end);
    
    updateArtifact(newValue);
    
    // Restore focus and selection
    setTimeout(() => {
      textarea.focus();
      const newCursorPos = start + before.length + (selectedText || 'text').length;
      textarea.setSelectionRange(newCursorPos, newCursorPos);
    }, 0);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Handle Cmd/Ctrl+B for bold
    if ((e.metaKey || e.ctrlKey) && e.key === 'b') {
      e.preventDefault();
      insertMarkdown('**', '**');
    }
    
    // Handle Cmd/Ctrl+I for italic
    if ((e.metaKey || e.ctrlKey) && e.key === 'i') {
      e.preventDefault();
      insertMarkdown('*', '*');
    }
    
    // Handle Cmd/Ctrl+S for save
    if ((e.metaKey || e.ctrlKey) && e.key === 's') {
      e.preventDefault();
      useCanvasStore.getState().saveArtifact();
    }
    
    // Handle Cmd/Ctrl+P for preview toggle
    if ((e.metaKey || e.ctrlKey) && e.key === 'p') {
      e.preventDefault();
      setIsPreview(!isPreview);
    }
  };

  if (!currentArtifact) return null;

  return (
    <div className="flex flex-col h-full bg-background-primary">
      {/* Markdown toolbar */}
      <div className="flex items-center gap-1 px-4 py-2 border-b border-border-subtle bg-background-secondary">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => insertMarkdown('**', '**')}
          title="Bold (Cmd+B)"
        >
          <Bold size={16} />
        </Button>
        
        <Button
          variant="ghost"
          size="sm"
          onClick={() => insertMarkdown('*', '*')}
          title="Italic (Cmd+I)"
        >
          <Italic size={16} />
        </Button>
        
        <div className="w-px h-4 bg-border-subtle mx-1" />
        
        <Button
          variant="ghost"
          size="sm"
          onClick={() => insertMarkdown('\n- ')}
          title="Bullet list"
        >
          <List size={16} />
        </Button>
        
        <Button
          variant="ghost"
          size="sm"
          onClick={() => insertMarkdown('\n1. ')}
          title="Numbered list"
        >
          <ListOrdered size={16} />
        </Button>
        
        <Button
          variant="ghost"
          size="sm"
          onClick={() => insertMarkdown('\n> ')}
          title="Quote"
        >
          <Quote size={16} />
        </Button>
        
        <div className="w-px h-4 bg-border-subtle mx-1" />
        
        <Button
          variant="ghost"
          size="sm"
          onClick={() => insertMarkdown('`', '`')}
          title="Inline code"
        >
          <Code size={16} />
        </Button>
        
        <Button
          variant="ghost"
          size="sm"
          onClick={() => insertMarkdown('[', '](url)')}
          title="Link"
        >
          <Link size={16} />
        </Button>
        
        <Button
          variant="ghost"
          size="sm"
          onClick={() => insertMarkdown('![alt text](', ')')}
          title="Image"
        >
          <Image size={16} />
        </Button>
        
        <div className="flex-1" />
        
        <Button
          variant={isPreview ? 'default' : 'ghost'}
          size="sm"
          onClick={() => setIsPreview(!isPreview)}
          title="Toggle preview (Cmd+P)"
        >
          {isPreview ? <Edit3 size={16} /> : <Eye size={16} />}
          <span className="ml-2">{isPreview ? 'Edit' : 'Preview'}</span>
        </Button>
      </div>

      {/* Content area */}
      <div className="flex-1 overflow-auto">
        {isPreview ? (
          <div className="px-6 py-4 prose prose-invert max-w-none">
            <ReactMarkdown
              components={{
                pre: ({ children }) => (
                  <pre className="bg-background-tertiary rounded-md p-3 overflow-x-auto">
                    {children}
                  </pre>
                ),
                code: ({ children, ...props }) => {
                  const isInline = !props.className?.includes('language-');
                  return isInline 
                    ? <code className="bg-background-tertiary px-1 py-0.5 rounded text-xs">{children}</code>
                    : <code>{children}</code>;
                },
              }}
            >
              {currentArtifact.content || '# Start writing...\n\nYour markdown content will appear here.'}
            </ReactMarkdown>
          </div>
        ) : (
          <textarea
            ref={textareaRef}
            value={currentArtifact.content}
            onChange={(e) => updateArtifact(e.target.value)}
            onKeyDown={handleKeyDown}
            className={cn(
              "w-full h-full px-6 py-4 bg-transparent resize-none outline-none",
              "text-foreground-primary placeholder-foreground-tertiary",
              wordWrap && "whitespace-pre-wrap break-words"
            )}
            style={{ 
              fontSize: `${fontSize}px`,
              lineHeight: '1.8',
              fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
            }}
            placeholder="# Start writing your document...

Use markdown formatting:
- **Bold** text with **double asterisks**
- *Italic* text with *single asterisks*
- Create lists with - or 1.
- Add [links](url) and ![images](url)
- Use `backticks` for inline code

Press Cmd+P to preview your document."
            spellCheck={true}
          />
        )}
      </div>
    </div>
  );
};