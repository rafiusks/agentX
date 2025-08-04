import { Zap, Grid, Code } from 'lucide-react'
import { useUIStore, UIMode } from '../../stores/ui.store'
import { useState, useRef, useEffect } from 'react'

export function ModeToggle() {
  const { mode, setMode } = useUIStore()
  const [isOpen, setIsOpen] = useState(false)
  const [showTooltip, setShowTooltip] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  
  const modes = [
    {
      id: 'simple' as UIMode,
      name: 'Simple',
      icon: Zap,
      description: 'Clean & focused'
    },
    {
      id: 'mission-control' as UIMode,
      name: 'Mission Control',
      icon: Grid,
      description: 'Advanced features'
    },
    {
      id: 'pro' as UIMode,
      name: 'Pro',
      icon: Code,
      description: 'Full power mode'
    }
  ]
  
  const currentMode = modes.find(m => m.id === mode)!
  const CurrentIcon = currentMode.icon
  
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])
  
  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        onMouseEnter={() => setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
        className="flex items-center gap-2 px-3 py-2 rounded-lg
                 bg-background-secondary border border-border-subtle
                 hover:border-border-default transition-all
                 text-sm relative group"
      >
        <CurrentIcon size={16} className="text-accent-blue" />
        <span className="text-foreground-primary font-medium">
          {currentMode.name}
        </span>
        {/* Power level indicator */}
        <div className="flex gap-0.5">
          <div className={`w-1 h-3 rounded-full transition-colors ${
            mode === 'simple' || mode === 'mission-control' || mode === 'pro' 
              ? 'bg-accent-blue' : 'bg-background-tertiary'
          }`} />
          <div className={`w-1 h-3 rounded-full transition-colors ${
            mode === 'mission-control' || mode === 'pro' 
              ? 'bg-accent-blue' : 'bg-background-tertiary'
          }`} />
          <div className={`w-1 h-3 rounded-full transition-colors ${
            mode === 'pro' 
              ? 'bg-accent-blue' : 'bg-background-tertiary'
          }`} />
        </div>
        <svg 
          width="12" 
          height="12" 
          viewBox="0 0 12 12" 
          fill="none" 
          className={`text-foreground-muted transition-transform ${isOpen ? 'rotate-180' : ''}`}
        >
          <path 
            d="M3 4.5L6 7.5L9 4.5" 
            stroke="currentColor" 
            strokeWidth="1.5" 
            strokeLinecap="round" 
            strokeLinejoin="round"
          />
        </svg>
        
        {/* Tooltip */}
        {showTooltip && !isOpen && (
          <div className="absolute top-full right-0 mt-2 px-3 py-2 
                        bg-background-tertiary border border-border-subtle rounded-lg
                        text-xs text-foreground-secondary whitespace-nowrap
                        pointer-events-none opacity-0 group-hover:opacity-100
                        transition-opacity duration-200 z-50">
            Switch interface complexity
            <div className="absolute bottom-full right-6 -translate-y-px">
              <div className="w-0 h-0 border-l-4 border-r-4 border-b-4 
                          border-l-transparent border-r-transparent 
                          border-b-background-tertiary" />
            </div>
          </div>
        )}
      </button>
      
      {isOpen && (
        <div className="absolute right-0 mt-2 w-64 rounded-lg
                      bg-background-secondary border border-border-subtle
                      shadow-xl z-50 overflow-hidden">
          <div className="p-2">
            {modes.map((modeOption) => {
              const Icon = modeOption.icon
              const isActive = mode === modeOption.id
              
              return (
                <button
                  key={modeOption.id}
                  onClick={() => {
                    setMode(modeOption.id)
                    setIsOpen(false)
                  }}
                  className={`
                    w-full flex items-center gap-3 px-3 py-2.5 rounded-md
                    transition-colors text-left
                    ${isActive 
                      ? 'bg-accent-blue/10 text-accent-blue' 
                      : 'hover:bg-background-tertiary text-foreground-primary'
                    }
                  `}
                >
                  <Icon size={18} className={isActive ? 'text-accent-blue' : 'text-foreground-secondary'} />
                  <div className="flex-1">
                    <div className="font-medium text-sm">{modeOption.name}</div>
                    <div className="text-xs text-foreground-muted">
                      {modeOption.description}
                    </div>
                  </div>
                  {isActive && (
                    <div className="w-2 h-2 rounded-full bg-accent-blue" />
                  )}
                </button>
              )
            })}
          </div>
          
          <div className="px-4 py-3 border-t border-border-subtle bg-background-tertiary">
            <p className="text-xs text-foreground-muted">
              <kbd className="px-1.5 py-0.5 bg-background-secondary rounded text-[10px] font-medium">⌘K</kbd>
              {' '}to quickly switch modes
            </p>
          </div>
        </div>
      )}
    </div>
  )
}