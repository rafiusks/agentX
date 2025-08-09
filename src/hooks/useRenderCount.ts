import { useRef, useEffect } from 'react';

/**
 * Debug hook to track component render counts
 */
export function useRenderCount(componentName: string) {
  const renderCount = useRef(0);
  const lastRenderTime = useRef(Date.now());
  
  renderCount.current++;
  
  useEffect(() => {
    const now = Date.now();
    const timeSinceLastRender = now - lastRenderTime.current;
    
    // Only log if rendering too frequently (more than once per 100ms)
    if (timeSinceLastRender < 100 && renderCount.current > 10) {
      console.warn(`[Performance] ${componentName} rendered ${renderCount.current} times. Last render was ${timeSinceLastRender}ms ago`);
    }
    
    lastRenderTime.current = now;
  });
  
  // Log every 50 renders to track excessive re-renders
  if (renderCount.current % 50 === 0) {
    console.warn(`[Performance] ${componentName} has rendered ${renderCount.current} times`);
  }
}