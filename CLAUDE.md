# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 开发约定 / Development Guidelines

- **语言**: 使用中文回答用户问题
- **开发环境**: Windows 系统
- **BAT 脚本**:
  - 脚本内容不要包含中文字符
  - 修复脚本时在原有脚本上修改，非必需不要新建脚本文件
- **图片识别**: 使用 MCP/Agent 截图能力前，确保图片尺寸小于 1000x1000
- **项目计划**: 计划文件存放于 `docs\plans\` 目录

## Project Overview

**AI-Commit** is a CLI tool that enhances Git workflows with AI-powered capabilities:
- Generate Conventional Commits-compliant messages using AI
- Run AI code reviews on staged diffs
- Review/enforce commit message style

Supports multiple AI providers: OpenAI, Google Gemini, Anthropic Claude, DeepSeek, Phind (free), Ollama (local), and OpenRouter.

## Build and Development

```bash
# Build from source
go build -o ai-commit ./cmd/ai-commit

# Build with version info
go build -ldflags "-X main.version=$(git describe --tags) -X main.commit=$(git rev-parse HEAD) -X main.date=$(date)" ./cmd/ai-commit

# Run tests (if any exist)
go test ./...

# Format code
go fmt ./...

# Vet code
go vet ./...
```

**Go version**: 1.25.0

## Architecture

### Core Abstractions

The codebase is organized around a few key abstractions that enable pluggable AI providers and a clean separation of concerns:

**Provider Registry Pattern** (`pkg/provider/registry/`):
- Central registry using factory pattern for AI providers
- `Register(name, Factory)` - registers a provider constructor
- `Get(name)` - retrieves provider factory
- `RegisterDefaults(name, ProviderSettings)` - sets default model/baseURL
- `SetRequiresAPIKey(name, bool)` - marks API key requirement
- Thread-safe with sync.RWMutex

**AI Client Interface** (`pkg/ai/ai.go`):
```go
type AIClient interface {
    GetCommitMessage(ctx context.Context, prompt string) (string, error)
    SanitizeResponse(message, commitType string) string
    ProviderName() string
    MaybeSummarizeDiff(diff string, maxLength int) (string, bool)
}

type StreamingAIClient interface {
    StreamCommitMessage(ctx context.Context, prompt string, onDelta func(delta string)) (final string, err error)
}
```
- `BaseAIClient` provides default implementations for `SanitizeResponse` and `MaybeSummarizeDiff`
- Streaming is optional - providers implement `StreamingAIClient` if supported

**Configuration Priority**:
```
CLI flags > Environment variables > Config file > Registry defaults
```

### Package Structure

| Package | Purpose |
|---------|---------|
| `cmd/ai-commit` | Main entry point using Cobra CLI |
| `pkg/ai` | AI client interface and base implementations |
| `pkg/config` | YAML config with validation (go-playground/validator) |
| `pkg/git` | Git operations wrapper around go-git |
| `pkg/provider` | AI provider implementations (openai, google, anthropic, etc.) |
| `pkg/provider/registry` | Provider factory registry |
| `pkg/ui` | Bubble Tea terminal UI with state machine |
| `pkg/prompt` | AI prompt construction |
| `pkg/committypes` | Conventional Commits type handling |
| `pkg/template` | Commit message templating |
| `pkg/summarizer` | Commit summarization |
| `pkg/versioner` | Semantic release |
| `pkg/splitter` | Interactive commit splitting |
| `pkg/httpx` | HTTP utilities and SSE handling |

### Key Design Patterns

1. **Provider Registration**: Each provider package has an `init()` function that registers itself with the registry and sets defaults/API key requirements

2. **Streaming Fallback**: The UI attempts streaming if available, falls back to non-streaming gracefully

3. **Lock File Filtering**: Diffs for files in `lockFiles` config (go.mod, go.sum, etc.) are excluded from AI prompts to reduce noise

4. **State Machine UI**: The TUI (`pkg/ui/`) uses distinct states (generating, committing, diff viewing, etc.) with Bubble Tea

### Configuration

Config file location: `~/.config/ai-commit/config.yaml` (path derives from binary name)

Key configuration sections:
- `authorName`/`authorEmail` - Git identity (tool does NOT read git config)
- `provider` - default provider
- `providers.<name>` - per-provider settings (apiKey, model, baseURL)
- `limits.diff/prompt` - truncation limits
- `commitTypes` - Conventional Commits types with emoji mappings
- `template` - commit message template with `{COMMIT_MESSAGE}` and `{GIT_BRANCH}` placeholders
- `promptTemplate` - global prompt template for AI operations

### Environment Variables

Format: `{PROVIDER}_API_KEY` and `{PROVIDER}_BASE_URL` (uppercase provider name)
Examples: `OPENAI_API_KEY`, `ANTHROPIC_BASE_URL`

## Provider Implementation Checklist

When adding a new AI provider:

1. Create package in `pkg/provider/<name>/`
2. Implement `AIClient` interface (embed `BaseAIClient` for defaults)
3. Optionally implement `StreamingAIClient` if supported
4. Add `init()` function to register with registry:
   ```go
   func init() {
       registry.Register("<name>", NewClient)
       registry.RegisterDefaults("<name>", config.ProviderSettings{...})
       registry.SetRequiresAPIKey("<name>", true/false)
   }
   ```
5. Import provider package in `cmd/ai-commit/main.go`

## Testing Approach

This project primarily uses manual/integration testing rather than unit tests. When making changes:
- Test with actual Git repos
- Test with multiple AI providers
- Verify TUI interactions
- Check both interactive and `--force` modes
