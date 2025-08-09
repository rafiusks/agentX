import { useState } from 'react';
import { ChevronDown, Zap, Brain, Eye, Code2, DollarSign, HardDrive } from 'lucide-react';

interface Model {
  id: string;
  name: string;
  provider: string;
  contextWindow: string;
  capabilities: string[];
  costPerMillion?: number;
  isLocal?: boolean;
  speed: 'fast' | 'medium' | 'slow';
}

interface ModelSelectorProps {
  currentModel: string;
  onModelChange: (modelId: string) => void;
  connectionId?: string;
}

const models: Model[] = [
  {
    id: 'gpt-4-turbo',
    name: 'GPT-4 Turbo',
    provider: 'OpenAI',
    contextWindow: '128K',
    capabilities: ['vision', 'code', 'functions'],
    costPerMillion: 10,
    speed: 'medium'
  },
  {
    id: 'gpt-3.5-turbo',
    name: 'GPT-3.5 Turbo',
    provider: 'OpenAI',
    contextWindow: '16K',
    capabilities: ['code', 'functions'],
    costPerMillion: 0.5,
    speed: 'fast'
  },
  {
    id: 'claude-3-opus',
    name: 'Claude 3 Opus',
    provider: 'Anthropic',
    contextWindow: '200K',
    capabilities: ['vision', 'code'],
    costPerMillion: 15,
    speed: 'medium'
  },
  {
    id: 'claude-3-sonnet',
    name: 'Claude 3 Sonnet',
    provider: 'Anthropic',
    contextWindow: '200K',
    capabilities: ['vision', 'code'],
    costPerMillion: 3,
    speed: 'fast'
  },
  {
    id: 'llama-3-70b',
    name: 'Llama 3 70B',
    provider: 'Local',
    contextWindow: '8K',
    capabilities: ['code'],
    isLocal: true,
    speed: 'slow'
  }
];

export const ModelSelector = ({
  currentModel,
  onModelChange
}: ModelSelectorProps) => {
  const [isOpen, setIsOpen] = useState(false);
  const [hoveredModel, setHoveredModel] = useState<string | null>(null);
  
  const selectedModel = models.find(m => m.id === currentModel) || models[0];
  
  const getCapabilityIcon = (capability: string) => {
    switch (capability) {
      case 'vision': return <Eye size={12} />;
      case 'code': return <Code2 size={12} />;
      case 'functions': return <Brain size={12} />;
      default: return null;
    }
  };
  
  const getSpeedColor = (speed: string) => {
    switch (speed) {
      case 'fast': return 'text-green-400';
      case 'medium': return 'text-yellow-400';
      case 'slow': return 'text-red-400';
      default: return 'text-foreground-muted';
    }
  };
  
  return (
    <div className="relative">
      {/* Current Model Display */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-2 bg-background-secondary hover:bg-background-tertiary 
                 rounded-xl border border-border-subtle/50 transition-all duration-200 group"
      >
        <div className="flex items-center gap-2">
          <Brain size={16} className="text-accent-blue" />
          <div className="text-left">
            <div className="text-sm font-medium text-foreground-primary">
              {selectedModel.name}
            </div>
            <div className="flex items-center gap-2 text-xs text-foreground-muted">
              <span>{selectedModel.contextWindow}</span>
              <span>•</span>
              <span className={getSpeedColor(selectedModel.speed)}>
                {selectedModel.speed}
              </span>
              {selectedModel.isLocal && (
                <>
                  <span>•</span>
                  <HardDrive size={10} className="text-green-400" />
                </>
              )}
            </div>
          </div>
        </div>
        <ChevronDown 
          size={16} 
          className={`text-foreground-muted transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`}
        />
      </button>
      
      {/* Dropdown */}
      {isOpen && (
        <>
          {/* Backdrop */}
          <div 
            className="fixed inset-0 z-40" 
            onClick={() => setIsOpen(false)}
          />
          
          {/* Model List */}
          <div className="absolute top-full mt-2 left-0 w-80 bg-background-secondary border border-border-subtle 
                        rounded-xl shadow-2xl z-50 overflow-hidden">
            <div className="p-2">
              <div className="text-xs text-foreground-muted px-3 py-2 font-semibold uppercase tracking-wider">
                Available Models
              </div>
              
              {models.map(model => (
                <button
                  key={model.id}
                  onClick={() => {
                    onModelChange(model.id);
                    setIsOpen(false);
                  }}
                  onMouseEnter={() => setHoveredModel(model.id)}
                  onMouseLeave={() => setHoveredModel(null)}
                  className={`
                    w-full text-left px-3 py-3 rounded-lg transition-all duration-150
                    ${model.id === currentModel 
                      ? 'bg-accent-blue/10 border border-accent-blue/30' 
                      : 'hover:bg-white/5 border border-transparent'
                    }
                  `}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-medium text-sm text-foreground-primary">
                          {model.name}
                        </span>
                        {model.id === currentModel && (
                          <span className="text-xs bg-accent-blue/20 text-accent-blue px-1.5 py-0.5 rounded">
                            Active
                          </span>
                        )}
                      </div>
                      
                      <div className="flex items-center gap-3 text-xs text-foreground-muted">
                        <span className="text-foreground-secondary">{model.provider}</span>
                        <span>•</span>
                        <span>{model.contextWindow} context</span>
                        <span>•</span>
                        <span className={getSpeedColor(model.speed)}>{model.speed}</span>
                      </div>
                      
                      <div className="flex items-center gap-3 mt-1.5">
                        {/* Capabilities */}
                        <div className="flex items-center gap-1">
                          {model.capabilities.map(cap => (
                            <div
                              key={cap}
                              className="p-1 bg-white/5 rounded"
                              title={cap}
                            >
                              {getCapabilityIcon(cap)}
                            </div>
                          ))}
                        </div>
                        
                        {/* Cost */}
                        {model.costPerMillion && (
                          <div className="flex items-center gap-1 text-xs text-foreground-muted">
                            <DollarSign size={10} />
                            <span>${model.costPerMillion}/M</span>
                          </div>
                        )}
                        
                        {/* Local indicator */}
                        {model.isLocal && (
                          <div className="flex items-center gap-1 text-xs text-green-400">
                            <HardDrive size={10} />
                            <span>Local</span>
                          </div>
                        )}
                      </div>
                    </div>
                    
                    {/* Speed Indicator */}
                    <div className="ml-4">
                      <Zap size={16} className={getSpeedColor(model.speed)} />
                    </div>
                  </div>
                  
                  {/* Hover Info */}
                  {hoveredModel === model.id && model.id !== currentModel && (
                    <div className="mt-2 pt-2 border-t border-border-subtle/30">
                      <span className="text-xs text-accent-blue">
                        Click to switch to this model
                      </span>
                    </div>
                  )}
                </button>
              ))}
            </div>
            
            {/* Footer */}
            <div className="px-4 py-3 border-t border-border-subtle/30 bg-background-tertiary/30">
              <div className="flex items-center justify-between text-xs">
                <span className="text-foreground-muted">
                  Token usage resets on model change
                </span>
                <button className="text-accent-blue hover:underline">
                  Manage API Keys
                </button>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
};