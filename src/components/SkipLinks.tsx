/**
 * Skip navigation links for keyboard and screen reader users
 */
export function SkipLinks() {
  return (
    <div className="sr-only focus-within:not-sr-only focus-within:absolute focus-within:top-4 focus-within:left-4 focus-within:z-50">
      <a
        href="#main-content"
        className="bg-accent-blue text-white px-4 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-accent-blue focus:ring-offset-2"
      >
        Skip to main content
      </a>
      <a
        href="#message-input"
        className="bg-accent-blue text-white px-4 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-accent-blue focus:ring-offset-2 ml-2"
      >
        Skip to message input
      </a>
      <a
        href="#chat-messages"
        className="bg-accent-blue text-white px-4 py-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-accent-blue focus:ring-offset-2 ml-2"
      >
        Skip to messages
      </a>
    </div>
  );
}