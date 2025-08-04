import { Plus, MessageSquare, Trash2 } from 'lucide-react'
import { useChatStore } from '../../stores/chat.store'
import * as ScrollArea from '@radix-ui/react-scroll-area'

export function ChatSidebar() {
  const { 
    sessions, 
    currentSessionId, 
    createSession, 
    setCurrentSession,
    deleteSession 
  } = useChatStore()

  return (
    <div className="w-64 border-r border-border-subtle bg-background-secondary/50">
      <div className="p-4">
        <button
          onClick={createSession}
          className="w-full btn-primary px-4 py-2 gap-2"
        >
          <Plus size={16} />
          New Chat
        </button>
      </div>
      
      <ScrollArea.Root className="flex-1 h-[calc(100%-80px)]">
        <ScrollArea.Viewport className="w-full h-full px-2">
          <div className="space-y-1 pb-4">
            {sessions.map(session => (
              <div
                key={session.id}
                className={`
                  group flex items-center gap-2 px-3 py-2 rounded-md
                  cursor-pointer transition-all
                  ${currentSessionId === session.id 
                    ? 'bg-background-tertiary text-foreground-primary' 
                    : 'hover:bg-background-tertiary/50 text-foreground-secondary'
                  }
                `}
                onClick={() => setCurrentSession(session.id)}
              >
                <MessageSquare size={14} />
                <span className="flex-1 text-sm truncate">
                  {session.title}
                </span>
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    deleteSession(session.id)
                  }}
                  className="opacity-0 group-hover:opacity-100 transition-opacity
                           p-1 rounded hover:bg-background-accent"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))}
          </div>
        </ScrollArea.Viewport>
        <ScrollArea.Scrollbar
          className="flex select-none touch-none p-0.5 bg-background-secondary transition-colors duration-[160ms] ease-out hover:bg-background-tertiary data-[orientation=vertical]:w-2.5 data-[orientation=horizontal]:flex-col data-[orientation=horizontal]:h-2.5"
          orientation="vertical"
        >
          <ScrollArea.Thumb className="flex-1 bg-foreground-muted rounded-[10px] relative before:content-[''] before:absolute before:top-1/2 before:left-1/2 before:-translate-x-1/2 before:-translate-y-1/2 before:w-full before:h-full before:min-w-[44px] before:min-h-[44px]" />
        </ScrollArea.Scrollbar>
      </ScrollArea.Root>
    </div>
  )
}