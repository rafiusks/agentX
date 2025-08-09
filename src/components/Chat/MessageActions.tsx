import { useState } from 'react';
import { Copy, RotateCcw, Edit3, Trash2, Share2, ThumbsUp, ThumbsDown, Check } from 'lucide-react';

interface MessageActionsProps {
  messageId: string;
  content: string;
  role: 'user' | 'assistant' | 'system' | 'function';
  onCopy?: () => void;
  onRegenerate?: () => void;
  onEdit?: (newContent: string) => void;
  onDelete?: () => void;
  onShare?: () => void;
}

export const MessageActions = ({
  messageId,
  content,
  role,
  onCopy,
  onRegenerate,
  onEdit,
  onDelete,
  onShare
}: MessageActionsProps) => {
  const [copied, setCopied] = useState(false);
  const [feedback, setFeedback] = useState<'up' | 'down' | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState(content);
  
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
      onCopy?.();
    } catch (error) {
      console.error('Failed to copy:', error);
    }
  };
  
  const handleFeedback = (type: 'up' | 'down') => {
    setFeedback(feedback === type ? null : type);
    // In a real implementation, this would send feedback to the server
    console.log(`Feedback for message ${messageId}: ${type}`);
  };
  
  const handleEdit = () => {
    if (isEditing && editContent !== content) {
      onEdit?.(editContent);
    }
    setIsEditing(!isEditing);
  };
  
  const handleRegenerate = () => {
    onRegenerate?.();
  };
  
  return (
    <>
      {/* Floating Action Bar */}
      <div className="message-actions-bar absolute -top-8 right-0 flex items-center gap-1 
                    bg-background-secondary border border-border-subtle 
                    rounded-xl px-2 py-1 shadow-lg opacity-0 group-hover:opacity-100 
                    transition-all duration-200 z-10">
        
        {/* Copy Button */}
        <button
          onClick={handleCopy}
          className="p-1.5 hover:bg-white/5 rounded transition-colors"
          title="Copy message"
        >
          {copied ? (
            <Check size={14} className="text-green-400" />
          ) : (
            <Copy size={14} className="text-foreground-muted" />
          )}
        </button>
        
        {/* Edit Button (for user messages) */}
        {role === 'user' && onEdit && (
          <button
            onClick={handleEdit}
            className={`p-1.5 hover:bg-white/5 rounded transition-colors ${
              isEditing ? 'bg-white/10 text-accent-blue' : ''
            }`}
            title="Edit message"
          >
            <Edit3 size={14} className="text-foreground-muted" />
          </button>
        )}
        
        {/* Regenerate Button (for assistant messages) */}
        {role === 'assistant' && onRegenerate && (
          <button
            onClick={handleRegenerate}
            className="p-1.5 hover:bg-white/5 rounded transition-colors"
            title="Regenerate response"
          >
            <RotateCcw size={14} className="text-foreground-muted" />
          </button>
        )}
        
        {/* Feedback Buttons (for assistant messages) */}
        {role === 'assistant' && (
          <>
            <div className="w-px h-4 bg-border-subtle/50 mx-1" />
            <button
              onClick={() => handleFeedback('up')}
              className={`p-1.5 hover:bg-white/5 rounded transition-colors ${
                feedback === 'up' ? 'text-green-400' : 'text-foreground-muted'
              }`}
              title="Good response"
            >
              <ThumbsUp size={14} />
            </button>
            <button
              onClick={() => handleFeedback('down')}
              className={`p-1.5 hover:bg-white/5 rounded transition-colors ${
                feedback === 'down' ? 'text-red-400' : 'text-foreground-muted'
              }`}
              title="Bad response"
            >
              <ThumbsDown size={14} />
            </button>
          </>
        )}
        
        {/* Share Button */}
        {onShare && (
          <>
            <div className="w-px h-4 bg-border-subtle/50 mx-1" />
            <button
              onClick={onShare}
              className="p-1.5 hover:bg-white/5 rounded transition-colors"
              title="Share message"
            >
              <Share2 size={14} className="text-foreground-muted" />
            </button>
          </>
        )}
        
        {/* Delete Button */}
        {onDelete && (
          <>
            <div className="w-px h-4 bg-border-subtle/50 mx-1" />
            <button
              onClick={onDelete}
              className="p-1.5 hover:bg-red-500/10 rounded transition-colors group/delete"
              title="Delete message"
            >
              <Trash2 size={14} className="text-foreground-muted group-hover/delete:text-red-400" />
            </button>
          </>
        )}
      </div>
      
      {/* Edit Mode */}
      {isEditing && (
        <div className="absolute inset-0 bg-background-primary rounded-2xl p-4 z-20 border border-border-subtle">
          <textarea
            value={editContent}
            onChange={(e) => setEditContent(e.target.value)}
            className="w-full h-full bg-background-secondary border border-border-subtle rounded-lg 
                     p-3 text-foreground-primary resize-none focus:outline-none 
                     focus:ring-2 focus:ring-accent-blue/50"
            autoFocus
          />
          <div className="absolute bottom-2 right-2 flex gap-2">
            <button
              onClick={() => {
                setIsEditing(false);
                setEditContent(content);
              }}
              className="px-3 py-1 text-sm text-foreground-muted hover:text-foreground-primary 
                       transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleEdit}
              className="px-3 py-1 text-sm bg-accent-blue text-white rounded-md 
                       hover:bg-accent-blue/90 transition-colors"
            >
              Save
            </button>
          </div>
        </div>
      )}
    </>
  );
};