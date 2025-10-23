# cc-plugins

A curated collection of Claude Code plugins for enhanced development workflows.

## Installation

Install this plugin marketplace in Claude Code by running:

```bash
/plugin marketplace add vaayne/cc-plugins
```

## Available Plugins

### specs-dev

Spec-driven feature development workflow with GPT-5 (Codex) review. Combines iterative requirements gathering, comprehensive planning, and structured implementation with continuous AI-powered code review. Produces production-ready code with proper planning documentation and clean, incremental commits.

**Commands:**
- `/spec:plan` - Collaborative planning workflow with Codex review before implementation
- `/spec:impl` - Implement feature from spec with iterative codex review and commits

## Development

This project uses [mise](https://mise.jdx.dev/) for dependency management and task automation.

### Prerequisites

Install mise:

```bash
# macOS
brew install mise

# Or use the official installer
curl https://mise.run | sh
```

### Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/vaayne/cc-plugins.git
   cd cc-plugins
   ```

2. Install dependencies with mise:
   ```bash
   mise install
   ```

3. Run available tasks:
   ```bash
   mise tasks
   ```

## Plugin Structure

```
.claude-plugin/
  marketplace.json       # Plugin marketplace metadata
plugins/
  specs-dev/            # Spec-driven development plugin
    README.md           # Plugin documentation
    commands/           # Slash commands
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT
