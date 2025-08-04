# AgentX Keyboard Shortcuts

## Global Shortcuts
These shortcuts work across all UI layers.

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Ctrl+Q` | Quit | Exit AgentX |
| `Ctrl+Shift+M` | Toggle Mission Control | Switch to/from Terminal UI (Mission Control) mode |
| `Ctrl+Shift+P` | Toggle Pro Mode | Switch to/from Professional (Advanced) mode |
| `Tab` | Model Selection | Open model selector to choose AI provider/model |
| `Ctrl+C` | Cancel | Cancel current operation |

## Simple UI Layer
Basic chat interface shortcuts.

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Enter` | Send Message | Submit your query to the AI |
| `Esc` | Clear Input | Clear the current input field |
| `Tab` | Model Selector | Switch between available models |
| `↑`/`↓` | History | Navigate through message history |

## Mission Control (Terminal) Layer
Warp-style terminal interface with command blocks.

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Enter` | Execute Command | Run the current command |
| `Ctrl+K` | Command Palette | Open the command palette |
| `Ctrl+L` | Clear Screen | Clear the terminal display |
| `Ctrl+Shift+L` | Clear History | Clear command history |
| `Ctrl+E` | Export | Export current session |
| `Ctrl+/` | Comment | Toggle comment for current line |
| `Tab` | Autocomplete | Complete command or show suggestions |
| `Ctrl+R` | Search History | Search through command history |
| `↑`/`↓` | Navigate | Move through command blocks |
| `Ctrl+↑`/`Ctrl+↓` | Jump Blocks | Jump between command blocks |

## Pro Mode Layer
Advanced development environment.

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Ctrl+Space` | AI Assist | Trigger AI assistance at cursor |
| `Ctrl+Shift+A` | Agent Panel | Toggle agent orchestration panel |
| `Ctrl+Shift+D` | Debug Panel | Toggle debug information |
| `Ctrl+Shift+M` | Metrics | Show performance metrics |
| `Ctrl+P` | Quick Open | Quick file/command search |
| `Ctrl+Shift+P` | Command Palette | Full command palette |
| `F1` | Help | Context-sensitive help |

## Model Selection Mode
Active when Tab is pressed.

| Shortcut | Action | Description |
|----------|--------|-------------|
| `↑`/`↓` | Navigate | Move through available models |
| `Enter` | Select | Choose the highlighted model |
| `Esc` | Cancel | Exit model selection |
| `/` | Filter | Start typing to filter models |
| `Space` | Toggle Details | Show/hide model details |

## Command Palette
Active when Ctrl+K is pressed.

| Shortcut | Action | Description |
|----------|--------|-------------|
| `↑`/`↓` | Navigate | Move through commands |
| `Enter` | Execute | Run selected command |
| `Esc` | Close | Close command palette |
| `Ctrl+Enter` | Execute & Close | Run command and close palette |
| Type to search | Filter | Filter commands as you type |

## Tips & Tricks

### Progressive Disclosure
- Start in Simple mode for basic chat
- AgentX automatically suggests Mission Control after 5 interactions
- Manually switch modes with `Ctrl+Shift+M` or `Ctrl+Shift+P`

### Model Switching
- Press `Tab` anytime to see available models
- Models show health status: 🟢 Online, 🔴 Offline, 🟡 Config Issue
- Recently used models appear at the top

### Command Blocks (Mission Control)
- Each command and its output forms a "block"
- Navigate blocks with `Ctrl+↑`/`Ctrl+↓`
- Copy entire blocks with `Ctrl+Shift+C`
- Share blocks with `Ctrl+Shift+S`

### AI Integration
- Natural language commands: Type plain English, AgentX translates
- Context awareness: AI understands your project structure
- Smart suggestions: Based on your command history

### MCP Servers
- Tab through MCP servers as additional providers
- MCP servers appear with 🔌 icon in model selector
- Configure in `~/.agentx/config.toml`

## Customization

Edit `~/.agentx/config.toml` to customize shortcuts:

```toml
[shortcuts]
quit = "ctrl+q"
model_select = "tab"
command_palette = "ctrl+k"
clear_screen = "ctrl+l"

[ui]
show_hints = true
vim_mode = false  # Coming soon!
```

## Accessibility

AgentX supports standard terminal accessibility features:
- Screen reader compatible
- High contrast themes available
- Keyboard-only navigation
- Customizable font sizes via terminal settings