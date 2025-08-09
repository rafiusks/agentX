import { useEffect, useRef } from 'react';

export function useInputDebug(componentName: string, value: string) {
  const renderCount = useRef(0);
  const lastRenderTime = useRef(performance.now());
  const inputEventTimes = useRef<number[]>([]);
  
  renderCount.current++;
  
  useEffect(() => {
    const now = performance.now();
    const timeSinceLastRender = now - lastRenderTime.current;
    
    // Track input events
    inputEventTimes.current.push(now);
    
    // Keep only last 10 events
    if (inputEventTimes.current.length > 10) {
      inputEventTimes.current.shift();
    }
    
    // Calculate average time between input events
    if (inputEventTimes.current.length > 1) {
      const deltas = [];
      for (let i = 1; i < inputEventTimes.current.length; i++) {
        deltas.push(inputEventTimes.current[i] - inputEventTimes.current[i - 1]);
      }
      const avgDelta = deltas.reduce((a, b) => a + b, 0) / deltas.length;
      
      // Warn if input events are slow
      if (avgDelta > 100) {
        console.warn(`[InputDebug] ${componentName} - Slow input detected:`, {
          avgTimeBetweenEvents: avgDelta.toFixed(2) + 'ms',
          lastRenderTime: timeSinceLastRender.toFixed(2) + 'ms',
          renderCount: renderCount.current,
          valueLength: value.length
        });
      }
    }
    
    lastRenderTime.current = now;
  }, [value, componentName]);
  
  // Log performance marks
  if (typeof window !== 'undefined' && window.performance) {
    performance.mark(`${componentName}-input-${value.length}`);
  }
}