import { useEffect, useRef } from 'react';
import { useStreamingStore } from '../../stores/streaming.store';

/**
 * Screen reader announcer for live regions
 * Announces important changes to screen reader users
 */
export function AccessibilityAnnouncer() {
  const { isStreaming, streamBuffer, streamError } = useStreamingStore();
  const announcerRef = useRef<HTMLDivElement>(null);
  const lastAnnouncedRef = useRef<string>('');
  
  // Announce streaming status changes
  useEffect(() => {
    if (!announcerRef.current) return;
    
    if (isStreaming && streamBuffer.length === 0) {
      announcerRef.current.textContent = 'AI is thinking...';
    } else if (isStreaming && streamBuffer.length > 0) {
      // Announce periodically during streaming
      const words = streamBuffer.split(' ');
      if (words.length % 10 === 0) { // Every 10 words
        announcerRef.current.textContent = `Receiving response... ${words.length} words so far`;
      }
    } else if (!isStreaming && lastAnnouncedRef.current !== 'complete') {
      announcerRef.current.textContent = 'Response complete';
      lastAnnouncedRef.current = 'complete';
    }
  }, [isStreaming, streamBuffer]);
  
  // Announce errors
  useEffect(() => {
    if (streamError && announcerRef.current) {
      announcerRef.current.textContent = `Error: ${streamError}`;
    }
  }, [streamError]);
  
  return (
    <>
      {/* Screen reader only announcements */}
      <div
        ref={announcerRef}
        className="sr-only"
        role="status"
        aria-live="polite"
        aria-atomic="true"
      />
      
      {/* Additional live region for urgent announcements */}
      <div
        className="sr-only"
        role="alert"
        aria-live="assertive"
        aria-atomic="true"
        id="urgent-announcer"
      />
    </>
  );
}