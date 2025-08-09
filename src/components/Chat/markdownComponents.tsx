import { CodeBlock } from './CodeBlock';

// Stable markdown component definitions to prevent re-creation
export const markdownComponents = {
  h1: ({ children }: any) => <h1 className="text-xl font-semibold mt-6 mb-3 text-foreground-primary">{children}</h1>,
  h2: ({ children }: any) => <h2 className="text-lg font-semibold mt-5 mb-2.5 text-foreground-primary">{children}</h2>,
  h3: ({ children }: any) => <h3 className="text-base font-semibold mt-4 mb-2 text-foreground-primary">{children}</h3>,
  p: ({ children }: any) => <p className="mb-4 text-foreground-primary">{children}</p>,
  ul: ({ children }: any) => <ul className="mb-4 ml-4 space-y-2">{children}</ul>,
  ol: ({ children }: any) => <ol className="mb-4 ml-4 space-y-2 list-decimal">{children}</ol>,
  li: ({ children }: any) => <li className="pl-2">{children}</li>,
  blockquote: ({ children }: any) => (
    <blockquote className="border-l-4 border-accent-blue/30 pl-4 py-2 my-4 bg-accent-blue/[0.03] italic text-foreground-secondary">
      {children}
    </blockquote>
  ),
  hr: () => <hr className="my-6 border-border-subtle/30" />,
  pre: ({ children }: any) => {
    // Let the code component handle the rendering
    return <pre className="m-0 p-0">{children}</pre>;
  },
  code: ({ className, children, ...props }: any) => {
    const match = /language-(\w+)/.exec(className || '');
    const language = match ? match[1] : undefined;
    const isInline = !className;
    
    // For code blocks (non-inline), render the CodeBlock component
    if (!isInline && match) {
      return (
        <CodeBlock 
          language={language}
          value={String(children).replace(/\n$/, '')}
        />
      );
    }
    
    // For inline code, render as simple styled code
    return (
      <code className="px-1.5 py-0.5 bg-gray-900/30 text-blue-400 rounded-md text-sm font-mono">
        {children}
      </code>
    );
  },
  strong: ({ children }: any) => <strong className="font-semibold text-foreground-primary">{children}</strong>,
  em: ({ children }: any) => <em className="italic">{children}</em>,
  a: ({ children, href }: any) => (
    <a href={href} className="text-accent-blue hover:text-accent-blue/80 underline underline-offset-2 transition-colors" target="_blank" rel="noopener noreferrer">
      {children}
    </a>
  ),
  table: ({ children }: any) => (
    <div className="overflow-x-auto my-4">
      <table className="w-full border-collapse">
        {children}
      </table>
    </div>
  ),
  thead: ({ children }: any) => (
    <thead className="bg-background-tertiary/50">
      {children}
    </thead>
  ),
  tbody: ({ children }: any) => (
    <tbody>
      {children}
    </tbody>
  ),
  tr: ({ children }: any) => (
    <tr className="border-b border-border-subtle/30 even:bg-background-secondary/30">
      {children}
    </tr>
  ),
  th: ({ children }: any) => (
    <th className="px-4 py-2 text-left font-semibold text-foreground-primary border border-border-subtle/50">
      {children}
    </th>
  ),
  td: ({ children }: any) => (
    <td className="px-4 py-2 border border-border-subtle/30">
      {children}
    </td>
  ),
};