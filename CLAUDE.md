# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AgentX is evolving from a unified ChatGPT/Claude UI into a full AI IDE for agentic software development. Built in Rust for performance, it follows an incremental development path:

**Base Layer** (Current Focus): A fast, elegant chat interface for multiple LLM providers:
- Support for OpenAI, Anthropic, and local models (Ollama, llama.cpp)
- Bring Your Own Keys (BYOK) model for cloud APIs
- Terminal integration inspired by Warp's innovations
- Claude Code SDK integration for advanced capabilities

**Evolution to AI IDE**: Beyond chat, AgentX will become a true AI IDE where agents do the building while humans do the thinking, featuring agent orchestration, task decomposition, and multi-modal development experiences.

## Architecture

### High-Level Structure
- **Terminal UI System**: Three-layer progressive interface (Simple → Mission Control → Pro Mode)
- **Agent Orchestration**: Task decomposition and coordination between external AI services
- **Intelligence Interfaces**: Abstractions for integrating with external AI providers
- **Infrastructure**: NATS messaging, vector databases, and execution sandboxes
- **External AI Integration**: MCP servers, local LLMs (Ollama, llama.cpp), and cloud APIs

### Key Modules
- `src/ui/`: Terminal UI with Ratatui (simple and terminal layers)
- `src/agents/`: Agent orchestration and coordination logic
- `src/intelligence/`: Interfaces and patterns for external AI integration
- `src/infrastructure/`: Core systems (messaging, storage, execution)
- `src/app/`: Main application logic and event handling

### AI Architecture Philosophy
AgentX acts as a conductor, not a performer. The intelligence module provides:
- Abstractions and interfaces for external AI services
- Context management and routing logic
- Response formatting and UI integration
- Pattern matching for intent detection
- NO internal AI/ML algorithm implementations

## Evolution Strategy

### Phase 1: Unified LLM Interface (Current)
**Goal**: Build the best terminal-based chat interface for multiple LLMs
- Solid chat UI with streaming responses
- API key management (BYOK model)
- Local model integration (Ollama, llama.cpp)
- Basic terminal features

### Phase 2: Warp-Inspired Terminal Integration
**Goal**: Revolutionize terminal + AI interaction
- Block-based command architecture
- AI-powered command search and autocomplete
- Natural language to command translation
- Command history with rich metadata
- Collaborative features (shareable blocks)

### Phase 3: Full AI IDE
**Goal**: Transform from chat to true agentic development
- Agent orchestration system
- Task decomposition and planning
- Multi-agent parallel execution
- Spatial development canvas
- Time-travel debugging

## Development Commands

### Building
```bash
cargo build              # Debug build
cargo build --release    # Optimized release build
```

### Running
```bash
cargo run                # Run the application
cargo run --example terminal_ui_demo    # Run terminal UI demo
cargo run --example ai_features_demo    # Run AI features demo
cargo run --example orchestrator_demo   # Run orchestrator demo
```

### Testing
```bash
cargo test               # Run all tests
cargo test --lib         # Run library tests only
cargo test --examples    # Test examples compilation
```

### Code Quality
```bash
cargo fmt                # Format code
cargo clippy             # Run linter
cargo check              # Quick compilation check
```

## Current State vs Future Vision

### What Exists Now
- ✅ Simple UI layer (functional, running in main.rs)
- ✅ Terminal UI components (built but not integrated)
- ✅ Basic intelligence interfaces
- ⚠️ Agent orchestration (partially implemented)
- ❌ LLM provider integrations (not connected)
- ❌ Warp-style features (components exist, not wired up)

### Immediate Goals (Phase 1 Completion)
1. Wire up LLM providers (OpenAI, Anthropic, Ollama)
2. Implement streaming responses in the UI
3. Add API key management system
4. Create unified chat experience
5. Basic terminal command execution

### Architecture Reality Check
The project has two parallel UI implementations:
1. **Simple layer** (currently active): Basic prompt/response interface
2. **Terminal layer** (built, not integrated): Warp-inspired blocks, command palette

The immediate task is to create a working chat interface before evolving to the full terminal experience.

### External AI Integration Strategy
AgentX will create a symbiotic relationship with external AI services:
- **MCP Servers**: Will handle specialized AI tasks (code generation, analysis, etc.)
- **Local LLMs**: Via Ollama or llama.cpp for privacy-conscious operations
- **Cloud APIs**: For advanced capabilities when needed
- **Intelligence Module**: Provides the glue layer to orchestrate these services

To complete the integration:
- Update `src/app/mod.rs` to use terminal layer instead of simple layer
- Implement MCP server connections in the intelligence module
- Create adapters for local LLM providers (Ollama, llama.cpp)
- Wire the orchestrator to route tasks to appropriate external services

### Key Design Decisions
- **Rust-only core**: Maximum performance, <50ms startup time
- **Progressive disclosure**: UI complexity reveals based on user expertise
- **Async everywhere**: Built on Tokio for concurrent agent operations
- **Zero-copy streaming**: Efficient handling of AI model responses

## Working with the Codebase

### Adding New Features
1. Check todo.md for current priorities and phase planning
2. Follow the existing module structure
3. Use async/await for all potentially blocking operations
4. Maintain the progressive UI philosophy

### Performance Targets
- Startup time: <50ms
- UI response: <10ms
- Stream latency: <100ms to first token
- Memory usage: <50MB baseline

### Common Patterns
- Event-driven architecture for UI interactions
- Message passing between agents via channels
- Trait-based abstractions for extensibility
- Lock-free data structures where possible

## Dependencies to Note
- **ratatui**: Terminal UI framework
- **crossterm**: Terminal manipulation
- **tokio**: Async runtime
- **serde**: Serialization
- **anyhow/thiserror**: Error handling

## External AI Service Integration

### MCP Server Architecture
MCP (Model Context Protocol) servers will provide specialized AI capabilities:
- **Code Generation Server**: Handles code creation and modification
- **Analysis Server**: Code review, bug detection, security scanning
- **Documentation Server**: Generates and maintains documentation
- **Test Server**: Creates and runs test suites

### Local LLM Integration
For privacy and offline capabilities:
- **Ollama**: Primary local model provider
- **llama.cpp**: Direct model execution for maximum performance
- **Model Router**: Intelligently selects between local and remote models

### Orchestration Flow
1. User input → AgentX UI
2. Intent detection → Intelligence module
3. Task decomposition → Agent orchestrator
4. Service selection → Route to appropriate MCP/LLM
5. Response aggregation → Format for UI display
6. Continuous feedback → Learn and improve routing

## Core Value Propositions

### Why AgentX?
1. **Unified LLM Experience**: One interface for all models (local + cloud)
2. **Terminal-First Development**: AI assistance where developers live
3. **Progressive Complexity**: Start simple (chat), evolve to AI IDE
4. **Open Source Freedom**: No vendor lock-in, full transparency
5. **Performance Focus**: <50ms startup, instant responses
6. **Privacy First**: BYOK model, local options, your data stays yours

### Key Differentiators
- **vs Cursor/Windsurf**: Open source, terminal-native, multi-provider
- **vs ChatGPT/Claude Web**: Integrated terminal, developer-focused, extensible
- **vs Continue.dev**: Standalone app, no IDE dependency, Rust performance
- **vs Warp**: AI-first from ground up, open source, LLM agnostic

## Next Steps

### Immediate Priorities (Phase 1)
1. Implement LLM provider adapters (OpenAI, Anthropic, Ollama)
2. Wire up streaming responses to the UI
3. Add API key configuration system
4. Create basic chat loop with model selection
5. Test with multiple providers

### Next Phase (Terminal Integration)
1. Integrate the Warp-inspired terminal UI
2. Implement block-based command history
3. Add natural language command translation
4. Wire up command palette to real actions

See todo.md for detailed task breakdown and progress tracking.