/**
 * Detects if text looks like code and wraps it in markdown code blocks if needed
 */
export function formatCodeInText(text: string): string {
  // If the text already has code blocks, return as is
  if (text.includes('```')) {
    return text;
  }

  // Common code patterns
  const codePatterns = [
    /^(import|from|export|const|let|var|function|class|def|if|for|while|return)\s+/gm,
    /\w+\s*=\s*\w+\(/gm, // function calls with assignment
    /\w+\(\s*['"`].*['"`]\s*\)/gm, // function calls with strings
    /\{\s*\n.*\n\s*\}/gm, // multi-line objects
    /\[\s*\n.*\n\s*\]/gm, // multi-line arrays
    /\w+\.\w+\(/gm, // method calls
    /#\s*(Load|Split|Store|Query|.*)/gm, // Python comments
    /DirectoryLoader|RecursiveCharacterTextSplitter|OpenAIEmbeddings|Chroma/g, // Common library names
  ];

  // Common language keywords
  const languageKeywords = {
    python: ['import', 'from', 'def', 'class', '__init__', 'self', 'return', 'if', 'elif', 'else', 'for', 'while', 'with', 'as', 'try', 'except', 'lambda'],
    javascript: ['const', 'let', 'var', 'function', 'return', 'if', 'else', 'for', 'while', 'class', 'export', 'import', 'async', 'await', '=>'],
    typescript: ['const', 'let', 'var', 'function', 'return', 'if', 'else', 'for', 'while', 'class', 'export', 'import', 'interface', 'type', 'async', 'await', '=>'],
    go: ['package', 'import', 'func', 'var', 'const', 'type', 'struct', 'interface', 'return', 'if', 'else', 'for', 'range', 'defer'],
    rust: ['fn', 'let', 'mut', 'const', 'struct', 'impl', 'trait', 'enum', 'match', 'if', 'else', 'for', 'while', 'return', 'use'],
  };

  // Check if the text looks like code
  const looksLikeCode = codePatterns.some(pattern => pattern.test(text));
  
  if (!looksLikeCode) {
    return text;
  }

  // Try to detect the language
  let detectedLanguage = 'plaintext';
  let maxMatches = 0;

  for (const [lang, keywords] of Object.entries(languageKeywords)) {
    const matches = keywords.filter(keyword => {
      const regex = new RegExp(`\\b${keyword}\\b`, 'g');
      return regex.test(text);
    }).length;

    if (matches > maxMatches) {
      maxMatches = matches;
      detectedLanguage = lang;
    }
  }

  // If we detected code, wrap it in a code block
  if (maxMatches > 2) {
    // Check if the code is all on one line but should have line breaks
    // Look for common patterns that indicate line breaks
    let formattedText = text;
    
    // Add line breaks after imports
    formattedText = formattedText.replace(/(import .+?(?:from|$))/g, '$1\n');
    
    // Add line breaks after common statement endings
    formattedText = formattedText.replace(/([;\}])\s*([a-zA-Z#])/g, '$1\n$2');
    
    // Add line breaks before comments
    formattedText = formattedText.replace(/\s+(#)/g, '\n$1');
    
    // Fix Python specific patterns
    if (detectedLanguage === 'python') {
      // Add line breaks after closing parentheses followed by new statements
      formattedText = formattedText.replace(/\)\s+(\w+\s*=)/g, ')\n$1');
      // Fix chained method calls
      formattedText = formattedText.replace(/\)\s+\./g, ').');
    }
    
    return `\`\`\`${detectedLanguage}\n${formattedText}\n\`\`\``;
  }

  return text;
}