# AgentX Terminal UI Design

A sophisticated, Warp-inspired terminal interface that combines elegant design with powerful AI-driven productivity features.

## Design Philosophy

### Core Principles
- **Clarity through Structure**: Every command and output lives in its own visual block
- **Ambient Intelligence**: AI features are seamlessly integrated, not intrusive
- **Progressive Disclosure**: Simple by default, powerful when needed
- **Delightful Interactions**: Smooth animations and thoughtful micro-interactions

## Visual Design System

### Color Palette

#### Dark Theme (Default)
```
Background Hierarchy:
- Primary:   #0D1117 (Deep space black)
- Secondary: #161B22 (Elevated surfaces)
- Tertiary:  #1E252E (Panel backgrounds)
- Accent:    #30363D (Interactive states)

Text Hierarchy:
- Primary:   #C9D1D9 (Main content)
- Secondary: #8B949E (Supporting text)
- Muted:     #58606A (Disabled/hints)
- Accent:    #58A6FF (Links/highlights)

Semantic Colors:
- Success: #57AB5A (Confirmations)
- Error:   #F85149 (Errors)
- Warning: #FBBC04 (Warnings)
- Info:    #58A6FF (Information)

Syntax Highlighting:
- Keywords:  #FF7B72 (Coral)
- Strings:   #79C0FF (Sky blue)
- Numbers:   #FFA657 (Orange)
- Comments:  #58606A (Gray)
- Functions: #D2A8FF (Purple)
- Variables: #79C0FF (Light blue)
```

### Typography
- **Font**: SF Mono or JetBrains Mono
- **Size Hierarchy**:
  - Command Input: 14px
  - Output Text: 13px
  - UI Labels: 12px
  - Timestamps: 11px

## Component Architecture

### 1. Command Blocks

Each command execution creates a distinct visual block:

```
┌─────────────────────────────────────────────────┐
│ ✓ 14:23:45                        125ms │ AI ▼ │
├─────────────────────────────────────────────────┤
│ $ git status --porcelain                        │
├─────────────────────────────────────────────────┤
│ M  src/main.rs                                  │
│ ?? docs/design.md                               │
└─────────────────────────────────────────────────┘
```

**Features:**
- Rounded corners with subtle border
- Status indicator (✓/✗) with timestamp
- Execution duration
- Collapsible output
- AI explanation badge when available
- Syntax-highlighted command
- Copy/share buttons on hover

### 2. Intelligent Command Input

Multi-line editor with AI-powered features:

```
┌─────────────────────────────────────────────────┐
│ $ cargo test --lib |                            │
│   --features "async"                            │
│                                                 │
│ → Run all library tests with async features ✨  │
└─────────────────────────────────────────────────┘
```

**Features:**
- Mode indicators (Shell/$, Python/>>>, JS/>, AI mode)
- Real-time syntax highlighting
- Multi-line editing with proper indentation
- AI suggestions appear below with → prefix
- Natural language mode toggle (Ctrl+N)
- Smart autocomplete based on context

### 3. Command Palette (Cmd+K)

Centered modal with fuzzy search:

```
╭─────────────────────────────────────────────────╮
│ 🔍 Search commands, workflows, and actions...   │
├─────────────────────────────────────────────────┤
│ ▶ ⏱  git status                           ⌘G S │
│   📄 Docker compose template                    │
│   🤖 Explain this error                    ⌘E  │
│   ⚡ Generate test cases                    ⌘T  │
│   ⭐ Deploy to production                       │
╰─────────────────────────────────────────────────╯
```

**Categories:**
- ⏱ Recent Commands
- ⭐ Saved Commands
- ⚡ Workflows
- 📄 Templates
- ⚙️ Actions
- 🤖 AI Features

### 4. Layout Modes

#### Compact (< 80 chars)
```
┌─────────────────┐
│    Blocks       │
│                 │
├─────────────────┤
│  Command Input  │
├─────────────────┤
│   Status Bar    │
└─────────────────┘
```

#### Standard (80-120 chars)
```
┌──────────────────┬────────┐
│     Blocks       │Sidebar │
│                  │        │
├──────────────────┤        │
│  Command Input   │        │
├──────────────────┴────────┤
│        Status Bar         │
└───────────────────────────┘
```

#### Wide (> 120 chars)
```
┌──────────────────────┬─────────────┐
│       Blocks         │ AI Assistant│
│                      │             │
├──────────────────────┤   Context   │
│   Command Input      │   Analysis  │
├──────────────────────┤             │
│     Status Bar       │ Suggestions │
└──────────────────────┴─────────────┘
```

## Interaction Patterns

### Keyboard Shortcuts

**Global:**
- `Cmd+K`: Open command palette
- `Cmd+L`: Clear all blocks
- `Cmd+/`: Toggle AI mode
- `Cmd+S`: Save current session
- `Cmd+Shift+C`: Copy block as markdown

**Navigation:**
- `↑/↓`: Navigate command history
- `Cmd+↑/↓`: Navigate between blocks
- `Space`: Toggle block collapse
- `Tab`: Accept AI suggestion

**Editing:**
- `Shift+Enter`: New line in command
- `Ctrl+A/E`: Beginning/End of line
- `Ctrl+W`: Delete word
- `Ctrl+U`: Clear line

### AI Integration Points

1. **Inline Suggestions**
   - Ghost text appears as you type
   - Natural language → command translation
   - Context-aware completions

2. **Error Diagnostics**
   - Automatic error detection in output
   - "Explain" button appears on errors
   - Suggested fixes with one-click apply

3. **Command Explanations**
   - Hover over any command for explanation
   - Break down complex pipelines
   - Show potential risks/warnings

4. **Smart Workflows**
   - "Generate tests for this output"
   - "Create script from history"
   - "Debug this error"

## Animation & Transitions

### Micro-interactions
- **Command execution**: Subtle pulse animation on block
- **Block creation**: Slide in from bottom with fade
- **Suggestions**: Smooth fade-in with slight vertical shift
- **Palette open**: Scale up from center with backdrop fade
- **Copy action**: Brief highlight flash
- **Collapse/Expand**: Smooth height transition

### Performance
- 60fps animations using GPU acceleration
- Lazy rendering for off-screen blocks
- Virtual scrolling for long sessions
- Debounced syntax highlighting

## Collaborative Features

### Share Menu
```
╭─────────────────────────────────╮
│      Share This Block           │
├─────────────────────────────────┤
│ 🔗 Copy Link                    │
│ 📋 Copy as Markdown             │
│ 💬 Share to Team                │
│ 🎬 Record as GIF               │
╰─────────────────────────────────╯
```

### Team Features
- Real-time cursor positions
- Shared debugging sessions
- Command templates library
- Team workflow repository

## Accessibility

- **High Contrast Mode**: Increased color differentiation
- **Screen Reader**: Semantic HTML with ARIA labels
- **Keyboard Navigation**: Full keyboard accessibility
- **Motion Preferences**: Respect reduce-motion settings
- **Font Scaling**: UI scales with system font size

## Implementation Details

### Technology Stack
- **Renderer**: Ratatui for terminal UI
- **Syntax Highlighting**: Tree-sitter integration
- **Fuzzy Search**: Skim algorithm
- **AI Integration**: Local LLM with streaming

### Performance Targets
- Command execution: < 50ms overhead
- Palette open: < 100ms
- Syntax highlighting: < 16ms per frame
- AI suggestions: < 200ms for first token

This design creates a terminal experience that feels both familiar to power users and approachable to newcomers, with AI features that enhance rather than overwhelm the core terminal workflow.