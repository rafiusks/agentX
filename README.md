# AgentX - AI IDE for Agentic Software Development

<div align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/React-20232A?style=for-the-badge&logo=react&logoColor=61DAFB" alt="React" />
  <img src="https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/TypeScript-007ACC?style=for-the-badge&logo=typescript&logoColor=white" alt="TypeScript" />
</div>

## 🚀 Overview

AgentX is a powerful web application that unifies access to multiple AI models (OpenAI, Anthropic, Ollama) in a single, elegant interface. Built with a Go backend for high performance and a React frontend for a modern user experience.

## ✨ Features

- **🎨 Beautiful Termius-Inspired UI**: Dark theme with sophisticated neutrals and smooth animations
- **🤖 Multi-Provider Support**: OpenAI (GPT-4/3.5), Anthropic (Claude 3), Ollama (local), and Demo mode
- **⚡ Real-time Streaming**: Watch AI responses as they're generated
- **🔐 Secure & Private**: BYOK (Bring Your Own Keys) - your API keys stay on your machine
- **🎯 Command Palette**: Quick access to all features with Cmd/Ctrl+K
- **💻 Cross-Platform**: Works on Windows, macOS, and Linux

## 🏃‍♂️ Quick Start

### Prerequisites

1. **Install dependencies**
   ```bash
   # Install Go (1.21+)
   # Visit https://go.dev/dl/
   
   # Install Node.js (v18+)
   # Visit https://nodejs.org
   
   # Install Docker (for PostgreSQL)
   # Visit https://www.docker.com/get-started
   ```

2. **Clone and setup**
   ```bash
   git clone https://github.com/community/agentx
   cd agentx
   ```

3. **Start the services**
   ```bash
   # Start PostgreSQL and backend
   docker-compose up -d
   
   # Or run backend locally
   cd agentx-backend
   go run cmd/server/main.go
   
   # In another terminal, start frontend
   npm install
   npm run dev
   ```

4. **Open the app**
   Navigate to http://localhost:1420 in your browser

### Terminal Mode

For a terminal-based experience:
```bash
npm run terminal
```

## 🔑 API Key Setup

AgentX uses a BYOK (Bring Your Own Keys) model. Add your API keys through the Settings tab or via environment variables:

```bash
# OpenAI
export OPENAI_API_KEY=sk-...

# Anthropic  
export ANTHROPIC_API_KEY=sk-ant-...

# Ollama (no key needed, just install)
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull llama2
```

## 🏗️ Architecture

```
agentx/
├── src/                    # React frontend
│   ├── components/         # UI components
│   ├── stores/            # State management
│   └── styles/            # Tailwind styles
├── src-tauri/             # Rust backend
│   ├── src/               # Core logic
│   │   ├── providers/     # LLM integrations
│   │   ├── ui/           # Terminal UI
│   │   └── agents/       # Agent orchestration
│   └── tauri.conf.json   # Tauri config
```

## 🎯 Roadmap

- [x] Phase 1: Unified LLM Interface ✅
- [ ] Phase 2: Warp-Inspired Terminal Integration
- [ ] Phase 3: Full AI IDE with Agent Orchestration
- [ ] Phase 4: Visual Agent Workflow Builder
- [ ] Phase 5: Self-Improving Agents

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) first.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- UI design inspired by [Termius](https://termius.com)
- Built with [Tauri](https://tauri.app) for native performance
- Terminal UI powered by [Ratatui](https://ratatui.rs)