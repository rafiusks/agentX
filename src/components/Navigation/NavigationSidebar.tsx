import React, { useState } from 'react';
import { 
  Menu, 
  ChevronLeft,
  MessageSquare, 
  Plus, 
  Search,
  Settings,
  Server,
  Boxes,
} from 'lucide-react';
import { useChatStore } from '@/stores/chat.store';
import { useChats, useCreateChat, useUpdateChat, useDeleteChat } from '@/hooks/queries/useChats';
import { useDefaultConnection } from '@/hooks/queries/useConnections';
import { UserMenu } from '@/components/Auth/UserMenu';
import { ChatListItem } from './ChatListItem';

interface NavigationSidebarProps {
  currentTab: 'chat' | 'agents' | 'mcp' | 'settings';
  onTabChange: (tab: 'chat' | 'agents' | 'mcp' | 'settings') => void;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
}

export const NavigationSidebar: React.FC<NavigationSidebarProps> = ({
  currentTab,
  onTabChange,
  isCollapsed,
  onToggleCollapse
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [showAllChats, setShowAllChats] = useState(false);
  
  const { currentChatId, setCurrentChatId } = useChatStore();
  const { data: chats = [] } = useChats();
  const { data: defaultConnection } = useDefaultConnection();
  const createChatMutation = useCreateChat();
  const updateChatMutation = useUpdateChat();
  const deleteChatMutation = useDeleteChat();
  
  // Filter chats based on search
  const filteredChats = chats.filter(chat => {
    const title = chat.Title || chat.title || '';
    return title.toLowerCase().includes(searchQuery.toLowerCase());
  });
  
  // Chats are already sorted by the query hook, so we just use them directly
  // Show only recent chats unless expanded
  const displayedChats = showAllChats ? filteredChats : filteredChats.slice(0, 5);
  
  const handleCreateChat = () => {
    createChatMutation.mutate({
      title: `Chat ${chats.length + 1}`,
      provider: defaultConnection?.provider || 'openai',
      model: defaultConnection?.settings?.model || 'gpt-3.5-turbo'
    }, {
      onSuccess: (newChat) => {
        const chatId = newChat.ID || newChat.id;
        if (chatId) {
          setCurrentChatId(chatId);
          onTabChange('chat');
        }
      }
    });
  };
  
  const handleChatSelect = (chatId: string) => {
    setCurrentChatId(chatId);
    onTabChange('chat');
  };
  
  const handleRenameChat = (chatId: string, newTitle: string) => {
    updateChatMutation.mutate({
      id: chatId,
      title: newTitle
    });
  };
  
  const handleDeleteChat = (chatId: string) => {
    deleteChatMutation.mutate(chatId, {
      onSuccess: () => {
        // If we deleted the current chat, select another one
        if (chatId === currentChatId) {
          const remainingChats = chats.filter(c => (c.ID || c.id) !== chatId);
          if (remainingChats.length > 0) {
            setCurrentChatId(remainingChats[0].ID || remainingChats[0].id || null);
          } else {
            setCurrentChatId(null);
          }
        }
      }
    });
  };
  
  const navigationItems = [
    { id: 'chat' as const, label: 'Chats', icon: MessageSquare },
    { id: 'agents' as const, label: 'Agents', icon: Boxes },
    { id: 'mcp' as const, label: 'MCP Servers', icon: Server },
    { id: 'settings' as const, label: 'Settings', icon: Settings },
  ];
  
  return (
    <>
      {/* Sidebar */}
      <aside 
        className={`
          flex flex-col h-full border-r border-border-subtle/50
          transition-all duration-300 ease-out relative
          bg-gradient-to-b from-[rgba(26,26,26,0.95)] to-[rgba(20,20,20,0.98)]
          ${isCollapsed ? 'w-0 overflow-hidden' : 'w-[260px] shadow-[2px_0_12px_rgba(0,0,0,0.3)]'}
        `}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-border-subtle/30 backdrop-blur-sm">
          <h1 className="text-lg font-semibold bg-gradient-to-r from-white to-white/80 bg-clip-text text-transparent">AgentX</h1>
          <button
            onClick={onToggleCollapse}
            className="p-1.5 hover:bg-white/5 rounded-lg transition-all duration-200 group"
            title="Collapse sidebar"
          >
            <ChevronLeft size={18} className="text-foreground-secondary group-hover:text-foreground-primary transition-colors" />
          </button>
        </div>
        
        {/* Search */}
        <div className="p-3">
          <div className="relative group">
            <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-foreground-muted group-focus-within:text-accent-blue transition-colors" />
            <input
              type="text"
              placeholder="Search chats..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-2.5 bg-black/20 hover:bg-black/30 rounded-xl text-sm
                       placeholder-foreground-muted focus:outline-none focus:ring-2 
                       focus:ring-accent-blue/30 focus:bg-black/40 transition-all duration-200
                       border border-white/5 focus:border-accent-blue/30"
            />
          </div>
        </div>
        
        {/* New Chat Button */}
        <div className="px-3 pb-3">
          <button
            onClick={handleCreateChat}
            disabled={createChatMutation.isPending}
            className="w-full bg-gradient-to-r from-accent-blue to-accent-blue/90 text-white 
                     hover:from-accent-blue/90 hover:to-accent-blue/80
                     px-4 py-2.5 gap-2 rounded-xl font-medium transition-all duration-200
                     active:scale-[0.97] flex items-center justify-center
                     shadow-lg shadow-accent-blue/20 hover:shadow-xl hover:shadow-accent-blue/30"
          >
            <Plus size={16} className="transition-transform group-hover:rotate-90" />
            New Chat
          </button>
        </div>
        
        {/* Main Content */}
        <div className="flex-1 overflow-y-auto overflow-x-hidden">
          <div className="p-3 space-y-6">
            {/* Recent Chats Section */}
            {currentTab === 'chat' && (
              <div>
                <h3 className="text-xs font-semibold text-foreground-muted/60 uppercase tracking-wider mb-3 px-3">
                  Conversations
                </h3>
                <div className="space-y-0.5">
                    {displayedChats.length === 0 ? (
                      <div className="text-center py-8 px-4">
                        <MessageSquare size={32} className="mx-auto mb-3 text-foreground-muted/30" />
                        <p className="text-sm text-foreground-muted">
                          {searchQuery ? 'No chats found' : 'No conversations yet'}
                        </p>
                        <p className="text-xs text-foreground-muted/60 mt-1">
                          Start a new chat to begin
                        </p>
                      </div>
                    ) : (
                      <>
                        {displayedChats.map((chat) => (
                          <ChatListItem
                            key={chat.ID || chat.id}
                            chat={{
                              ...chat,
                              updatedAt: chat.UpdatedAt || chat.CreatedAt
                            }}
                            isActive={(chat.ID || chat.id) === currentChatId}
                            onSelect={handleChatSelect}
                            onRename={handleRenameChat}
                            onDelete={handleDeleteChat}
                          />
                        ))}
                        
                        {filteredChats.length > 5 && !showAllChats && (
                          <button
                            onClick={() => setShowAllChats(true)}
                            className="w-full text-left px-3 py-2 text-sm text-accent-blue
                                     hover:bg-background-tertiary/50 rounded-lg transition-colors"
                          >
                            See {filteredChats.length - 5} more...
                          </button>
                        )}
                        
                        {showAllChats && filteredChats.length > 5 && (
                          <button
                            onClick={() => setShowAllChats(false)}
                            className="w-full text-left px-3 py-2 text-sm text-accent-blue
                                     hover:bg-background-tertiary/50 rounded-lg transition-colors"
                          >
                            Show less
                          </button>
                        )}
                      </>
                    )}
                  </div>
              </div>
            )}
          </div>
        </div>
        
        {/* Bottom Section */}
        <div className="border-t border-border-subtle/30">
          {/* Navigation Items */}
          <div className="p-3 space-y-1">
            {navigationItems.map(item => {
              const Icon = item.icon;
              const isActive = currentTab === item.id;
              
              return (
                <button
                  key={item.id}
                  onClick={() => onTabChange(item.id)}
                  className={`
                    w-full text-left px-3 py-2.5 rounded-xl text-sm
                    transition-all duration-200 flex items-center gap-3 relative
                    ${isActive 
                      ? 'bg-accent-blue/10 text-accent-blue' 
                      : 'hover:bg-white/5 text-foreground-secondary hover:text-foreground-primary'
                    }
                  `}
                >
                  {isActive && (
                    <div className="absolute inset-0 bg-accent-blue/10 rounded-xl" />
                  )}
                  <Icon size={16} className="relative z-10" />
                  <span className="relative z-10">{item.label}</span>
                </button>
              );
            })}
          </div>
          
          {/* User Menu */}
          <div className="p-3 border-t border-border-subtle/30">
            <UserMenu />
          </div>
        </div>
      </aside>
      
      {/* Collapse Toggle Button (Always Visible) */}
      {isCollapsed && (
        <button
          onClick={onToggleCollapse}
          className="fixed left-4 top-4 z-50 p-2.5 bg-background-secondary/95 backdrop-blur-sm
                   hover:bg-background-tertiary rounded-xl border border-border-subtle/50
                   shadow-lg hover:shadow-xl transition-all duration-200 group"
          title="Open sidebar (⌘/)"
        >
          <Menu size={18} className="text-foreground-secondary group-hover:text-foreground-primary transition-colors" />
        </button>
      )}
    </>
  );
};