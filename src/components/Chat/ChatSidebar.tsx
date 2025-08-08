import { Plus, MessageSquare, Trash2 } from 'lucide-react'
import { useChatStore } from '../../stores/chat.store'
import { useChats, useCreateChat, useDeleteChat } from '../../hooks/queries/useChats'
import { useDefaultConnection } from '../../hooks/queries/useConnections'
import * as ScrollArea from '@radix-ui/react-scroll-area'

export function ChatSidebar() {
  const { 
    currentChatId,
    setCurrentChatId
  } = useChatStore()
  
  const { data: chats = [] } = useChats()
  const { data: defaultConnection } = useDefaultConnection()
  const createChatMutation = useCreateChat()
  const deleteChatMutation = useDeleteChat()
  
  console.log('ChatSidebar - chats:', chats)

  const handleCreateChat = () => {
    createChatMutation.mutate({
      title: `Chat ${chats.length + 1}`,
      provider: defaultConnection?.provider || 'openai',
      model: defaultConnection?.settings?.model || 'gpt-3.5-turbo'
    }, {
      onSuccess: (newChat) => {
        const chatId = newChat.ID || newChat.id;
        if (chatId) {
          setCurrentChatId(chatId)
        }
      }
    })
  }

  const handleDeleteChat = (chatId: string) => {
    deleteChatMutation.mutate(chatId)
    
    // If deleting current chat, select another
    if (chatId === currentChatId) {
      const remainingChats = chats.filter(c => c.id !== chatId)
      if (remainingChats.length > 0) {
        setCurrentChatId(remainingChats[0].id || remainingChats[0].ID || null)
      } else {
        setCurrentChatId(null)
      }
    }
  }

  return (
    <div className="w-64 h-full border-r border-border-subtle bg-background-secondary/50 flex flex-col">
      <div className="p-4">
        <button
          onClick={handleCreateChat}
          disabled={createChatMutation.isPending}
          className="w-full bg-accent-blue text-white hover:bg-accent-blue/90 px-4 py-2 gap-2 rounded-lg font-medium transition-all active:scale-[0.98] flex items-center justify-center"
        >
          <Plus size={16} />
          New Chat
        </button>
      </div>
      
      <ScrollArea.Root className="flex-1 h-[calc(100%-80px)]">
        <ScrollArea.Viewport className="w-full h-full px-2">
          <div className="space-y-1 pb-4">
            {(!chats || chats.length === 0) ? (
              <div className="text-center py-8">
                <p className="text-sm text-foreground-tertiary">No chats yet</p>
                <p className="text-xs text-foreground-tertiary mt-1">Click "New Chat" to start</p>
              </div>
            ) : (
              (chats || []).map((chat, _index) => {
              const chatId = chat.ID || chat.id;
              const chatTitle = chat.Title || chat.title;
              
              if (!chat || !chatId) return null;
              
              return (
                <div
                  key={chatId}
                  className={`
                    group flex items-center gap-2 px-3 py-2 rounded-md
                    cursor-pointer transition-all
                    ${currentChatId === chatId 
                      ? 'bg-background-tertiary text-foreground-primary' 
                      : 'hover:bg-background-tertiary/50 text-foreground-secondary'
                    }
                  `}
                  onClick={() => setCurrentChatId(chatId)}
                >
                  <MessageSquare size={14} />
                  <span className="flex-1 text-sm truncate">
                    {chatTitle || 'Untitled Chat'}
                  </span>
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDeleteChat(chatId)
                    }}
                    className="opacity-0 group-hover:opacity-100 transition-opacity
                             p-1 hover:bg-background-primary rounded"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              );
            })
            )}
          </div>
        </ScrollArea.Viewport>
        <ScrollArea.Scrollbar
          className="flex touch-none select-none bg-background-secondary p-0.5 transition-colors duration-150 ease-out hover:bg-background-tertiary data-[orientation=vertical]:w-2.5 data-[orientation=horizontal]:flex-col data-[orientation=horizontal]:h-2.5"
          orientation="vertical"
        >
          <ScrollArea.Thumb className="relative flex-1 rounded-full bg-foreground-tertiary before:absolute before:left-1/2 before:top-1/2 before:h-full before:min-h-[44px] before:w-full before:min-w-[44px] before:-translate-x-1/2 before:-translate-y-1/2 before:content-['']" />
        </ScrollArea.Scrollbar>
        <ScrollArea.Corner className="bg-background-secondary" />
      </ScrollArea.Root>
    </div>
  )
}