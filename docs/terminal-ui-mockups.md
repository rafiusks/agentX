# AgentX Terminal UI Visual Mockups

## Main Terminal View

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ AgentX Terminal                                              ⚙ 👤 ⊞ ⊟ ✕    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ ✓ 14:23:45                                      125ms │ AI │ ⋯ │ ▼ │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ $ git status --porcelain                                            │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ M  src/main.rs                                                      │   │
│ │ M  src/ui/terminal/mod.rs                                          │   │
│ │ ?? docs/design.md                                                   │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ ✓ 14:24:12                                      2.5s │      │ ⋯ │ ▼ │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ $ cargo build --release                                             │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │    Compiling agentx v0.1.0 (/Users/Rafael.Vidal/Code/agentX)      │   │
│ │     Finished release [optimized] target(s) in 2.48s                │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ ✗ 14:24:45                                      156ms │ AI │ ⋯ │ ▼ │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ $ cargo test                                                        │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ running 2 tests                                                     │   │
│ │ test tests::test_basic ... ok                                      │   │
│ │ test tests::test_advanced ... FAILED                               │   │
│ │                                                                     │   │
│ │ failures:                                                           │   │
│ │     tests::test_advanced                                           │   │
│ │                                                                     │   │
│ │ test result: FAILED. 1 passed; 1 failed; 0 ignored                │   │
│ │                                                                     │   │
│ │ 💡 AI: The test is failing due to an assertion error. The         │   │
│ │     expected value doesn't match the actual output. Click to       │   │
│ │     see suggested fix...                                           │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ $ create a web server with user authentication                      │   │
│ │                                                                     │   │
│ │ → cargo new auth-server && cd auth-server && cargo add actix-web  │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ Normal │ git:main │ 3 blocks                          AI Assistant Online  │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Command Palette View (Cmd+K)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│                   ╭─────────────────────────────────────────╮              │
│                   │                                         │              │
│                   │  Command Palette                        │              │
│                   │                                         │              │
│                   │  🔍 deploy to prod|                    │              │
│                   ├─────────────────────────────────────────┤              │
│                   │                                         │              │
│                   │  ▶ ⭐ Deploy to production         ⌘D  │              │
│                   │    Push current branch to production   │              │
│                   │                                         │              │
│                   │    ⚡ Production deployment workflow    │              │
│                   │    Full CI/CD pipeline with rollback   │              │
│                   │                                         │              │
│                   │    🤖 Fix production issues            │              │
│                   │    AI analyzes logs and suggests fixes │              │
│                   │                                         │              │
│                   │    ⏱  kubectl get pods -n production  │              │
│                   │    Check production pod status         │              │
│                   │                                         │              │
│                   ╰─────────────────────────────────────────╯              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## AI Assistant Panel (Wide Mode)

```
┌──────────────────────────────────────────┬─────────────────────────────────┐
│ Terminal                                 │ AI Assistant                    │
│                                          │                                 │
│ ┌────────────────────────────────────┐  │ 🤖 Context Analysis            │
│ │ ✗ 14:24:45              156ms │ AI │  │                                 │
│ ├────────────────────────────────────┤  │ You have a failing test in     │
│ │ $ cargo test                       │  │ your test suite. The assertion │
│ │ test tests::test_advanced FAILED   │  │ on line 45 is comparing:       │
│ └────────────────────────────────────┘  │                                 │
│                                          │ Expected: "Hello, World!"       │
│ ┌────────────────────────────────────┐  │ Actual:   "Hello World"         │
│ │ $ fix the failing test|            │  │                                 │
│ │                                    │  │ The issue is a missing comma.   │
│ │ → Let me analyze and fix...       │  │                                 │
│ └────────────────────────────────────┘  │ ─────────────────────────────── │
│                                          │                                 │
│                                          │ 💡 Suggested Actions            │
│                                          │                                 │
│                                          │ 1. Fix assertion:               │
│                                          │    [Apply Fix]                  │
│                                          │                                 │
│                                          │ 2. Update test data:            │
│                                          │    [View Changes]               │
│                                          │                                 │
│                                          │ 3. Run tests again:             │
│                                          │    [Run cargo test]             │
│                                          │                                 │
│                                          │ ─────────────────────────────── │
│                                          │                                 │
│                                          │ 📚 Related Documentation        │
│                                          │                                 │
│                                          │ • Rust testing best practices  │
│                                          │ • Assert macros guide          │
│                                          │ • Test organization patterns   │
└──────────────────────────────────────────┴─────────────────────────────────┘
```

## Collaborative Session View

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ AgentX Terminal - Shared Session                      👥 Rafael, Alice, Bob │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ ✓ 14:23:45 Rafael                               125ms │    │ ⋯ │ ▼ │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ $ docker-compose up -d                                              │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ Creating network "agentx_default" with the default driver           │   │
│ │ Creating agentx_db_1    ... done                                    │   │
│ │ Creating agentx_redis_1 ... done                                    │   │
│ │ Creating agentx_app_1   ... done                                    │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ Alice is typing...                                                  │   │
│ │ $ docker ps|                                                        │   │
│ │   █                                                                 │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ 💬 Bob: "Let me check the logs for the app container"              │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ Collaborative │ git:main │ 3 users connected          Share: agentx.io/s/x7y│
└─────────────────────────────────────────────────────────────────────────────┘
```

## Error State with AI Assistance

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ ✗ 14:30:22                                      45ms │ AI │ ⋯ │ ▼ │    │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ $ npm run build                                                     │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ ERROR in ./src/components/App.tsx                                   │   │
│ │ Module not found: Error: Can't resolve './Header' in               │   │
│ │ '/Users/Rafael.Vidal/Code/agentX/src/components'                   │   │
│ │                                                                     │   │
│ │ ╭───────────────────────────────────────────────────────────╮      │   │
│ │ │ 🤖 AI Analysis                                            │      │   │
│ │ │                                                           │      │   │
│ │ │ The import './Header' is failing because the file        │      │   │
│ │ │ doesn't exist at that path. I found a similar file:      │      │   │
│ │ │                                                           │      │   │
│ │ │ • ./components/Header/Header.tsx                         │      │   │
│ │ │                                                           │      │   │
│ │ │ Would you like me to:                                    │      │   │
│ │ │ [Fix Import Path] [Create Header.tsx] [Show More]        │      │   │
│ │ ╰───────────────────────────────────────────────────────────╯      │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Natural Language Input Mode

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ AI  refactor this function to use async/await instead of callbacks  │   │
│ │     and add proper error handling|                                  │   │
│ │                                                                     │   │
│ │ → I'll help you refactor the function. First, let me analyze       │   │
│ │   the current implementation and suggest the async/await version... │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Theme Variations

### Light Theme Preview
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ ┌─────────────────────────────────────────────────────────────────────┐   │
│ │ ✓ 14:23:45                                      125ms │    │ ⋯ │ ▼ │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ $ ls -la                                                            │   │
│ ├─────────────────────────────────────────────────────────────────────┤   │
│ │ drwxr-xr-x  12 rafael  staff   384 Jan 15 14:23 .                 │   │
│ │ drwxr-xr-x  20 rafael  staff   640 Jan 15 10:15 ..                │   │
│ │ -rw-r--r--   1 rafael  staff  1234 Jan 15 14:20 Cargo.toml        │   │
│ └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘

Light theme uses:
- Background: #FFFFFF, #F6F8FA, #F0F2F5
- Text: #24292E, #586069, #6A737D
- Accents: #0366D6, #28A745, #D73A49
```

## Gesture and Animation Guide

### Command Block Animations
- **Creation**: Slide up + fade in (200ms ease-out)
- **Collapse**: Height transition (150ms ease-in-out)
- **Selection**: Border glow pulse (300ms)
- **Copy**: Success flash (100ms)

### Command Palette
- **Open**: Scale from 0.95 → 1.0 + opacity 0 → 1 (150ms)
- **Close**: Reverse animation (100ms)
- **Item hover**: Background fade (50ms)
- **Search**: Instant filter with smooth list morph

### AI Suggestions
- **Appear**: Fade in + slide down 2px (200ms)
- **Accept**: Morph into command (100ms)
- **Dismiss**: Fade out (150ms)

### Scroll Behavior
- **Smooth scroll**: 60fps with momentum
- **Auto-scroll**: When new blocks appear
- **Snap points**: Align to block boundaries