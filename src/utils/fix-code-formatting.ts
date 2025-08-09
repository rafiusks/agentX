/**
 * Fix code formatting in LLM responses
 * Detects code that should be in code blocks and fixes it
 */

export function fixCodeFormatting(text: string): string {
  // If already has proper code blocks, return as is
  if (text.includes('```')) {
    return text;
  }
  
  // Split text into lines
  const lines = text.split('\n');
  const processedLines: string[] = [];
  let inCodeBlock = false;
  let codeBuffer: string[] = [];
  let codeLanguage = '';
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmedLine = line.trim();
    
    // Detect code start patterns
    const isCodeStart = 
      trimmedLine.startsWith('from ') ||
      trimmedLine.startsWith('import ') ||
      trimmedLine.startsWith('const ') ||
      trimmedLine.startsWith('let ') ||
      trimmedLine.startsWith('var ') ||
      trimmedLine.startsWith('function ') ||
      trimmedLine.startsWith('class ') ||
      trimmedLine.startsWith('def ') ||
      trimmedLine.startsWith('package ') ||
      trimmedLine.startsWith('interface ') ||
      trimmedLine.startsWith('type ') ||
      trimmedLine.match(/^\w+\s*=\s*[\[\{]/) || // Variable assignment to array/object
      trimmedLine.match(/^(client|response|tools|data|result)\s*=/) || // Common variable names
      (trimmedLine.includes('=') && trimmedLine.includes('[') && trimmedLine.includes('{'));
    
    // Detect if we're likely in a code block (indented or continuation)
    const looksLikeCode = 
      line.startsWith('  ') || // Indented
      line.startsWith('\t') || // Tab indented
      trimmedLine.startsWith('}') ||
      trimmedLine.startsWith(']') ||
      trimmedLine.startsWith(')') ||
      trimmedLine === '' && inCodeBlock; // Empty line in code
    
    if (isCodeStart && !inCodeBlock) {
      // Start a new code block
      if (processedLines.length > 0 && processedLines[processedLines.length - 1] !== '') {
        processedLines.push(''); // Add spacing before code block
      }
      
      // Detect language
      if (trimmedLine.includes('from ') || trimmedLine.includes('import ') && !trimmedLine.includes('{')) {
        codeLanguage = 'python';
      } else if (trimmedLine.includes('const ') || trimmedLine.includes('let ') || trimmedLine.includes('=>')) {
        codeLanguage = 'javascript';
      } else if (trimmedLine.includes('interface ') || trimmedLine.includes('type ')) {
        codeLanguage = 'typescript';
      } else if (trimmedLine.includes('package ') || trimmedLine.includes('func ')) {
        codeLanguage = 'go';
      } else {
        codeLanguage = 'python'; // Default to Python for this example
      }
      
      processedLines.push('```' + codeLanguage);
      codeBuffer = [line];
      inCodeBlock = true;
    } else if (inCodeBlock && looksLikeCode) {
      // Continue code block
      codeBuffer.push(line);
    } else if (inCodeBlock && !looksLikeCode && trimmedLine !== '') {
      // End code block if we hit non-code text
      processedLines.push(...codeBuffer);
      processedLines.push('```');
      processedLines.push('');
      processedLines.push(line);
      inCodeBlock = false;
      codeBuffer = [];
      codeLanguage = '';
    } else if (inCodeBlock && trimmedLine === '') {
      // Empty line in code block
      codeBuffer.push(line);
    } else {
      // Regular text
      if (inCodeBlock) {
        // Close any open code block
        processedLines.push(...codeBuffer);
        processedLines.push('```');
        processedLines.push('');
        inCodeBlock = false;
        codeBuffer = [];
        codeLanguage = '';
      }
      processedLines.push(line);
    }
  }
  
  // Close any remaining code block
  if (inCodeBlock && codeBuffer.length > 0) {
    processedLines.push(...codeBuffer);
    processedLines.push('```');
  }
  
  return processedLines.join('\n');
}

/**
 * Detect if text contains unformatted code
 */
export function hasUnformattedCode(text: string): boolean {
  // Already has code blocks
  if (text.includes('```')) {
    return false;
  }
  
  // Check for code patterns
  const codePatterns = [
    /^from\s+\w+\s+import/m,
    /^import\s+\w+/m,
    /^(const|let|var)\s+\w+\s*=/m,
    /^function\s+\w+/m,
    /^class\s+\w+/m,
    /^def\s+\w+/m,
    /client\s*=\s*\w+\(\)/,
    /response\s*=\s*\w+\.[\w.]+\(/,
    /tools\s*=\s*\[/,
  ];
  
  return codePatterns.some(pattern => pattern.test(text));
}