import { useEffect, useRef, useState } from 'react';

/**
 * Hook to debounce streaming updates to reduce re-renders
 * Batches updates and only triggers re-renders at intervals
 */
export function useDebouncedStream<T>(
  value: T,
  delay: number = 50
): T {
  const [debouncedValue, setDebouncedValue] = useState(value);
  const timerRef = useRef<ReturnType<typeof setTimeout>>();
  const accumulatedRef = useRef(value);

  useEffect(() => {
    // Update accumulated value immediately
    accumulatedRef.current = value;

    // Clear existing timer
    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }

    // Set new timer to update debounced value
    timerRef.current = setTimeout(() => {
      setDebouncedValue(accumulatedRef.current);
    }, delay);

    // Cleanup on unmount or when value becomes null/undefined
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        // Immediately update on cleanup to ensure final value is shown
        setDebouncedValue(accumulatedRef.current);
      }
    };
  }, [value, delay]);

  // Return immediately if not streaming
  if (!value) return value;

  return debouncedValue;
}