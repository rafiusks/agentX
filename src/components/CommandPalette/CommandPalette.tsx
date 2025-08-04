import { useEffect, useState } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Command } from 'cmdk'
import { 
  Search, 
  Settings, 
  MessageSquare, 
  Boxes, 
  FileText,
  Zap,
  Grid,
  Code
} from 'lucide-react'
import { useUIStore } from '../../stores/ui.store'

interface CommandPaletteProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSettingsOpen: () => void
}

export function CommandPalette({ open, onOpenChange, onSettingsOpen }: CommandPaletteProps) {
  const [search, setSearch] = useState('')
  const { setMode } = useUIStore()

  useEffect(() => {
    if (!open) {
      setSearch('')
    }
  }, [open])

  const commands = [
    {
      category: 'Navigation',
      items: [
        { 
          id: 'chat', 
          label: 'Go to Chat', 
          icon: MessageSquare,
          onSelect: () => {
            // Handle navigation
            onOpenChange(false)
          }
        },
        { 
          id: 'agents', 
          label: 'Go to Agents', 
          icon: Boxes,
          onSelect: () => {
            // Handle navigation
            onOpenChange(false)
          }
        },
        { 
          id: 'settings', 
          label: 'Open Settings', 
          icon: Settings,
          onSelect: () => {
            onSettingsOpen()
            onOpenChange(false)
          }
        },
      ]
    },
    {
      category: 'Actions',
      items: [
        { 
          id: 'new-chat', 
          label: 'New Chat', 
          icon: MessageSquare,
          onSelect: () => {
            // Handle new chat
            onOpenChange(false)
          }
        },
        { 
          id: 'clear-chats', 
          label: 'Clear All Chats', 
          icon: FileText,
          onSelect: () => {
            // Handle clear chats
            onOpenChange(false)
          }
        },
      ]
    },
    {
      category: 'Interface Mode',
      items: [
        { 
          id: 'mode-simple', 
          label: 'Simple Mode', 
          icon: Zap,
          onSelect: () => {
            setMode('simple')
            onOpenChange(false)
          }
        },
        { 
          id: 'mode-mission-control', 
          label: 'Mission Control Mode', 
          icon: Grid,
          onSelect: () => {
            setMode('mission-control')
            onOpenChange(false)
          }
        },
        { 
          id: 'mode-pro', 
          label: 'Pro Mode', 
          icon: Code,
          onSelect: () => {
            setMode('pro')
            onOpenChange(false)
          }
        },
      ]
    },
    {
      category: 'Quick Actions',
      items: [
        { 
          id: 'quick-explain', 
          label: 'Explain Code', 
          icon: Zap,
          onSelect: () => {
            // Handle quick action
            onOpenChange(false)
          }
        },
        { 
          id: 'quick-refactor', 
          label: 'Refactor Code', 
          icon: Zap,
          onSelect: () => {
            // Handle quick action
            onOpenChange(false)
          }
        },
      ]
    }
  ]

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 backdrop-blur-sm animate-in" />
        <Dialog.Content className="fixed left-[50%] top-[20%] max-h-[60vh] w-[90vw] max-w-[600px] 
                                 translate-x-[-50%] overflow-hidden rounded-lg 
                                 bg-background-secondary border border-border-subtle shadow-2xl
                                 animate-slide-up">
          <Command className="overflow-hidden">
            <div className="flex items-center border-b border-border-subtle px-4">
              <Search className="mr-2 h-4 w-4 text-foreground-muted" />
              <Command.Input
                placeholder="Type a command or search..."
                value={search}
                onValueChange={setSearch}
                className="flex h-12 w-full bg-transparent py-3 text-sm outline-none
                         placeholder:text-foreground-muted"
              />
            </div>
            
            <Command.List className="max-h-[400px] overflow-y-auto p-2">
              <Command.Empty className="py-6 text-center text-sm text-foreground-muted">
                No results found.
              </Command.Empty>
              
              {commands.map(category => (
                <Command.Group key={category.category} heading={category.category}>
                  <div className="px-2 py-1.5 text-xs font-medium text-foreground-muted">
                    {category.category}
                  </div>
                  {category.items.map(item => {
                    const Icon = item.icon
                    return (
                      <Command.Item
                        key={item.id}
                        value={item.label}
                        onSelect={item.onSelect}
                        className="flex items-center gap-3 rounded-md px-3 py-2
                                 text-sm text-foreground-primary
                                 hover:bg-background-tertiary cursor-pointer
                                 data-[selected=true]:bg-background-tertiary"
                      >
                        <Icon size={16} className="text-foreground-secondary" />
                        <span>{item.label}</span>
                      </Command.Item>
                    )
                  })}
                </Command.Group>
              ))}
            </Command.List>
            
            <div className="border-t border-border-subtle px-4 py-2">
              <div className="flex items-center gap-4 text-xs text-foreground-muted">
                <div className="flex items-center gap-1">
                  <kbd className="px-1.5 py-0.5 bg-background-tertiary rounded text-[10px]">↑↓</kbd>
                  <span>Navigate</span>
                </div>
                <div className="flex items-center gap-1">
                  <kbd className="px-1.5 py-0.5 bg-background-tertiary rounded text-[10px]">↵</kbd>
                  <span>Select</span>
                </div>
                <div className="flex items-center gap-1">
                  <kbd className="px-1.5 py-0.5 bg-background-tertiary rounded text-[10px]">esc</kbd>
                  <span>Close</span>
                </div>
              </div>
            </div>
          </Command>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}