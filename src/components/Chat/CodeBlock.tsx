import { useState, memo } from 'react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { Copy, Check, Terminal, Code2, FileText, Maximize2 } from 'lucide-react';

interface CodeBlockProps {
  language?: string;
  value: string;
  inline?: boolean;
}

const languageIcons: Record<string, React.ReactNode> = {
  javascript: <FileText size={12} className="text-yellow-400" />,
  typescript: <FileText size={12} className="text-blue-400" />,
  python: <FileText size={12} className="text-green-400" />,
  rust: <FileText size={12} className="text-orange-400" />,
  go: <FileText size={12} className="text-cyan-400" />,
  bash: <Terminal size={12} className="text-gray-400" />,
  shell: <Terminal size={12} className="text-gray-400" />,
  sql: <FileText size={12} className="text-purple-400" />,
  json: <Code2 size={12} className="text-yellow-300" />,
  html: <FileText size={12} className="text-red-400" />,
  css: <FileText size={12} className="text-blue-300" />,
  jsx: <FileText size={12} className="text-pink-400" />,
  tsx: <FileText size={12} className="text-pink-400" />,
};

const customStyle = {
  ...vscDarkPlus,
  'pre[class*="language-"]': {
    ...vscDarkPlus['pre[class*="language-"]'],
    background: 'transparent',
    margin: 0,
    padding: '1rem',
    fontSize: '14px',
    lineHeight: '1.6',
    overflow: 'auto',
    whiteSpace: 'pre',
  },
  'code[class*="language-"]': {
    ...vscDarkPlus['code[class*="language-"]'],
    background: 'transparent',
    fontSize: '14px',
    whiteSpace: 'pre',
  },
};

export const CodeBlock = memo(({ language, value, inline }: CodeBlockProps) => {
  const [copied, setCopied] = useState(false);
  const [isExpanded, setIsExpanded] = useState(false);
  
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('Failed to copy:', error);
    }
  };
  
  const lines = value.split('\n').length;
  const shouldCollapse = lines > 30 && !isExpanded;
  const displayValue = shouldCollapse ? value.split('\n').slice(0, 30).join('\n') : value;
  
  if (inline) {
    return (
      <code className="message-inline-code">
        {value}
      </code>
    );
  }
  
  return (
    <div className="message-code-block group relative my-4" style={{ maxWidth: '100%', overflow: 'hidden' }}>
      {/* Header Bar */}
      <div className="flex items-center justify-between px-4 py-2 bg-gray-900/70 border-b border-gray-800">
        <div className="flex items-center gap-2">
          {languageIcons[language || ''] || <Code2 size={12} className="text-gray-400" />}
          <span className="text-xs text-gray-400 font-mono">
            {language || 'plaintext'}
          </span>
          {lines > 1 && (
            <span className="text-xs text-gray-500">
              • {lines} lines
            </span>
          )}
        </div>
        
        <div className="flex items-center gap-1">
          {shouldCollapse && (
            <button
              onClick={() => setIsExpanded(!isExpanded)}
              className="p-1.5 hover:bg-gray-800 rounded transition-colors"
              title={isExpanded ? "Collapse" : "Expand"}
            >
              <Maximize2 size={14} className="text-gray-400" />
            </button>
          )}
          
          <button
            onClick={handleCopy}
            className="p-1.5 hover:bg-gray-800 rounded transition-colors"
            title="Copy code"
          >
            {copied ? (
              <Check size={14} className="text-green-400" />
            ) : (
              <Copy size={14} className="text-gray-400" />
            )}
          </button>
        </div>
      </div>
      
      {/* Code Content */}
      <div 
        className="code-scroll-container relative" 
        style={{ maxWidth: '100%' }}
      >
        <SyntaxHighlighter
          language={language || 'plaintext'}
          style={customStyle}
          showLineNumbers={lines > 5}
          wrapLines={false}
          wrapLongLines={false}
          customStyle={{
            background: 'rgba(17, 24, 39, 0.5)',
            margin: 0,
            borderRadius: 0,
            fontSize: '14px',
            minWidth: 'max-content',
          }}
          lineNumberStyle={{
            minWidth: '3em',
            paddingRight: '1em',
            color: '#4B5563',
            userSelect: 'none',
          }}
        >
          {displayValue}
        </SyntaxHighlighter>
        
        {/* Fade overlay for collapsed code */}
        {shouldCollapse && (
          <div className="absolute bottom-0 left-0 right-0 h-16 bg-gradient-to-t from-gray-900/90 to-transparent 
                        flex items-end justify-center pb-2 pointer-events-none">
            <button
              onClick={() => setIsExpanded(true)}
              className="text-xs text-accent-blue hover:text-accent-blue/80 pointer-events-auto"
            >
              Show more ({lines - 30} lines hidden)
            </button>
          </div>
        )}
      </div>
      
      {/* Line count indicator */}
      {lines > 10 && (
        <div className="absolute -left-12 top-1/2 -translate-y-1/2 hidden lg:flex items-center">
          <div className="text-xs text-gray-600 -rotate-90 whitespace-nowrap">
            {lines} lines
          </div>
        </div>
      )}
    </div>
  );
});

CodeBlock.displayName = 'CodeBlock';