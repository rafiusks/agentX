import React, { useState } from 'react';
import { Trash2, Edit3, Pin, Sparkles, MoreHorizontal } from 'lucide-react';
import { apiClient } from '../../lib/api-client';
import { useChatStore } from '@/stores/chat.store';

interface ChatListItemProps {
  chat: {
    id?: string;
    ID?: string;
    title?: string;
    Title?: string;
    updatedAt?: string;
    UpdatedAt?: string;
    isPinned?: boolean;
  };
  isActive: boolean;
  onSelect: (chatId: string) => void;
  onDelete?: (chatId: string) => void;
  onRename?: (chatId: string, newTitle: string) => void;
  onPin?: (chatId: string) => void;
}

export const ChatListItem: React.FC<ChatListItemProps> = ({
  chat,
  isActive,
  onSelect,
  onDelete,
  onRename,
  onPin
}) => {
  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [showMenu, setShowMenu] = useState(false);
  
  const { currentConnectionId } = useChatStore();
  
  const chatId = chat.ID || chat.id || '';
  const chatTitle = chat.Title || chat.title || 'Untitled';
  const chatUpdatedAt = chat.UpdatedAt || chat.updatedAt;
  
  // Generate title using AI
  const handleGenerateTitle = async () => {
    if (!chatId) return;
    
    setIsGenerating(true);
    try {
      console.log('[ChatListItem] Generating title for session:', chatId, 'with connection:', currentConnectionId);
      
      // Use the new LLM title generation endpoint
      const response = await apiClient.post<{ 
        title: string;
        provider?: string;
        connection_id?: string;
        duration_ms?: number;
      }>('/llm/tasks/title-generation', {
        session_id: chatId,
        connection_id: currentConnectionId || undefined // Optional parameter
      });
      
      if (response.title) {
        setEditTitle(response.title);
      }
    } catch (error) {
      console.error('Failed to generate title:', error);
      // Fallback to simple extraction if LLM fails
      setEditTitle(chatTitle);
    } finally {
      setIsGenerating(false);
    }
  };
  
  // Format timestamp
  const formatTime = (timestamp?: string) => {
    if (!timestamp) return '';
    
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    
    // Handle invalid dates
    if (isNaN(date.getTime())) {
      console.warn('Invalid timestamp:', timestamp);
      return '';
    }
    
    const minutes = Math.floor(diff / (1000 * 60));
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(hours / 24);
    
    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days === 1) return 'Yesterday';
    if (days < 7) return `${days}d ago`;
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  };
  
  const handleRename = () => {
    if (editTitle.trim() && onRename) {
      onRename(chatId, editTitle.trim());
      setIsEditing(false);
    }
  };
  
  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleRename();
    } else if (e.key === 'Escape') {
      setIsEditing(false);
      setEditTitle('');
    }
  };
  
  if (isEditing) {
    return (
      <div className="px-2 py-2 space-y-2">
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={editTitle}
            onChange={(e) => setEditTitle(e.target.value)}
            onKeyDown={handleKeyPress}
            placeholder="Enter chat title..."
            className="flex-1 px-3 py-1.5 bg-black/30 rounded-lg text-sm
                     focus:outline-none focus:ring-2 focus:ring-accent-blue/50
                     border border-white/10"
            autoFocus
            disabled={isGenerating}
          />
          <button
            onClick={handleGenerateTitle}
            disabled={isGenerating}
            className="p-1.5 bg-accent-blue/10 hover:bg-accent-blue/20 rounded-lg
                     transition-colors disabled:opacity-50 disabled:cursor-not-allowed
                     border border-accent-blue/20"
            title="Generate title with AI"
          >
            <Sparkles size={14} className={`text-accent-blue ${isGenerating ? 'animate-pulse' : ''}`} />
          </button>
        </div>
        <div className="flex gap-2 justify-end">
          <button
            onClick={() => {
              setIsEditing(false);
              setEditTitle('');
            }}
            className="px-3 py-1 text-xs text-foreground-secondary hover:text-foreground-primary
                     transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleRename}
            disabled={!editTitle.trim()}
            className="px-3 py-1 text-xs bg-accent-blue/20 hover:bg-accent-blue/30
                     text-accent-blue rounded-md transition-colors
                     disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Save
          </button>
        </div>
      </div>
    );
  }
  
  return (
    <div
      className={`
        relative group cursor-pointer
        ${isActive ? 'before:absolute before:left-0 before:top-2 before:bottom-2 before:w-[3px] before:bg-accent-blue before:rounded-r-full' : ''}
      `}
    >
      <div
        onClick={() => chatId && onSelect(chatId)}
        className={`
          relative flex items-center px-3 py-2 rounded-lg cursor-pointer
          transition-all duration-200 group
          ${isActive 
            ? 'bg-white/[0.08] text-foreground-primary ml-1' 
            : 'hover:bg-white/[0.04] text-foreground-secondary hover:text-foreground-primary'
          }
        `}
      >
        <h4 className="text-sm font-medium truncate flex-1 mr-8">
          {chat.isPinned && <Pin size={12} className="inline mr-1 text-accent-blue" />}
          {chatTitle}
        </h4>
        
        {/* Action button - positioned absolutely */}
        <div className="absolute right-2 top-1/2 -translate-y-1/2
                      opacity-0 group-hover:opacity-100 transition-opacity">
            <button
              onClick={(e) => {
                e.stopPropagation();
                setShowMenu(!showMenu);
              }}
              className="p-1 rounded text-foreground-muted hover:text-foreground-primary
                       transition-colors"
              title="More options"
            >
              <MoreHorizontal size={16} />
            </button>
            
          </div>
      </div>
      
      {/* Dropdown menu - positioned outside main content */}
      {showMenu && (
        <>
          {/* Click outside overlay */}
          <div 
            className="fixed inset-0 z-[100]" 
            onClick={() => setShowMenu(false)}
          />
          
          {/* Menu */}
          <div 
            className="absolute right-2 top-10 w-48 rounded-lg py-1 z-[101]"
            style={{ 
              backgroundColor: 'rgb(30, 30, 30)',
              border: '1px solid rgb(60, 60, 60)',
              boxShadow: '0 4px 12px rgba(0, 0, 0, 0.5)'
            }}
          >
            <button
              onClick={(e) => {
                e.stopPropagation();
                setEditTitle(chatTitle);
                setIsEditing(true);
                setShowMenu(false);
              }}
              className="w-full text-left px-3 py-2 text-sm text-gray-200 hover:bg-gray-700 transition-colors flex items-center gap-2"
            >
              <Edit3 size={14} />
              Rename
            </button>
            
            {onDelete && (
              <>
                <div className="border-t border-gray-700 my-1" />
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    if (window.confirm(`Delete "${chatTitle}"?`)) {
                      onDelete(chatId);
                      setShowMenu(false);
                    }
                  }}
                  className="w-full text-left px-3 py-2 text-sm text-red-400 hover:bg-gray-700 transition-colors flex items-center gap-2"
                >
                  <Trash2 size={14} />
                  Delete
                </button>
              </>
            )}
          </div>
        </>
      )}
    </div>
  );
};