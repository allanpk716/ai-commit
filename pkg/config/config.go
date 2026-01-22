package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

const (
    DefaultProvider         = "phind"
)

var (
	DefaultAuthorName  = "ai-commit"
	DefaultAuthorEmail = "ai-commit@example.com"
)

type CommitTypeConfig struct {
    Type  string `yaml:"type,omitempty"`
    Emoji string `yaml:"emoji,omitempty"`
}

// ProviderSettings holds credentials and routing for a provider.
type ProviderSettings struct {
    APIKey  string `yaml:"apiKey,omitempty"`
    Model   string `yaml:"model,omitempty"`
    BaseURL string `yaml:"baseURL,omitempty"`
}

type LimitSettings struct {
    Enabled  bool `yaml:"enabled,omitempty"`
    MaxChars int  `yaml:"maxChars,omitempty"`
}

type Limits struct {
    Diff   LimitSettings `yaml:"diff,omitempty"`
    Prompt LimitSettings `yaml:"prompt,omitempty"`
}

type PromptFiles struct {
    CommitMessage string `yaml:"commitMessage,omitempty"`
    CodeReview    string `yaml:"codeReview,omitempty"`
    StyleReview   string `yaml:"styleReview,omitempty"`
}

type Config struct {
	Prompt           string             `yaml:"prompt,omitempty"`
	CommitType       string             `yaml:"commitType,omitempty"`
	Template         string             `yaml:"template,omitempty"`
	SemanticRelease  bool               `yaml:"semanticRelease,omitempty"`
	InteractiveSplit bool               `yaml:"interactiveSplit,omitempty"`
	EnableEmoji      bool               `yaml:"enableEmoji,omitempty"`
	Language         string             `yaml:"language,omitempty"`

    Provider    string             `yaml:"provider,omitempty"`
    CommitTypes []CommitTypeConfig `yaml:"commitTypes,omitempty"`
    LockFiles   []string           `yaml:"lockFiles,omitempty"`
    Limits Limits `yaml:"limits,omitempty"`

    // Enterprise-style provider configuration. Preferred over legacy flat fields below.
    Providers map[string]ProviderSettings `yaml:"providers,omitempty"`

    // Prompt files configuration
    Prompts PromptFiles `yaml:"prompts,omitempty"`

    // Deprecated: Use Prompts.CommitMessage instead
    PromptTemplate string `yaml:"promptTemplate,omitempty"`

	AuthorName  string `yaml:"authorName,omitempty"`
	AuthorEmail string `yaml:"authorEmail,omitempty"`
}

func LoadOrCreateConfig() (*Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to determine executable path: %w", err)
	}
	binaryName := filepath.Base(exePath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine user home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", binaryName)
	configPath := filepath.Join(configDir, "config.yaml")

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}
	}

    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        defaultCfg := &Config{
            Provider:      DefaultProvider,
            Language:      "english",
            AuthorName:    DefaultAuthorName,
            AuthorEmail:   DefaultAuthorEmail,
            LockFiles:     []string{"go.mod", "go.sum"},
            Limits: Limits{
                Diff:   LimitSettings{Enabled: false, MaxChars: 0},
                Prompt: LimitSettings{Enabled: false, MaxChars: 0},
            },
            CommitTypes: []CommitTypeConfig{
                {Type: "feat", Emoji: "✨"},
                {Type: "fix", Emoji: "🐛"},
                {Type: "docs", Emoji: "📚"},
                {Type: "style", Emoji: "💎"},
                {Type: "refactor", Emoji: "♻️"},
                {Type: "test", Emoji: "🧪"},
                {Type: "chore", Emoji: "🔧"},
                {Type: "perf", Emoji: "🚀"},
                {Type: "build", Emoji: "📦"},
                {Type: "ci", Emoji: "👷"},
            },
            Providers: map[string]ProviderSettings{},
            Prompts: PromptFiles{
                CommitMessage: "commit-message.txt",
                CodeReview:    "code-review.txt",
                StyleReview:   "style-review.txt",
            },
            PromptTemplate: "",
        }
        if err := saveConfig(configPath, defaultCfg); err != nil {
            return nil, fmt.Errorf("failed to create default config: %w", err)
        }
        // Ensure prompt files exist after creating default config
        if err := ensurePromptFiles(configDir); err != nil {
            return nil, fmt.Errorf("failed to ensure prompt files: %w", err)
        }
        return defaultCfg, nil
    }

    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config file: %w", err)
    }
    // Ensure prompt files exist after loading config
    if err := ensurePromptFiles(configDir); err != nil {
        return nil, fmt.Errorf("failed to ensure prompt files: %w", err)
    }
    return &cfg, nil
}

func saveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func ResolveAPIKey(flagVal, envVar, configVal, provider string) (string, error) {
	if strings.TrimSpace(flagVal) != "" {
		return strings.TrimSpace(flagVal), nil
	}
	if envVal := os.Getenv(envVar); strings.TrimSpace(envVal) != "" {
		return strings.TrimSpace(envVal), nil
	}
	if strings.TrimSpace(configVal) != "" {
		return strings.TrimSpace(configVal), nil
	}
 
	return "", fmt.Errorf("%s API key is required. Provide via flag, %s environment variable, or config", provider, envVar)
}

func (cfg *Config) Validate() error {
    v := validator.New()
    if err := v.Struct(cfg); err != nil {
        return fmt.Errorf("config validation failed: %w", err)
    }
    return nil
}

// GetProviderSettings fetches settings from the Providers map and fills defaults.
func (cfg *Config) GetProviderSettings(name string) ProviderSettings {
    if cfg.Providers != nil {
        if ps, ok := cfg.Providers[name]; ok {
            return ps
        }
    }
    return ProviderSettings{}
}

// ensurePromptFiles creates the prompts directory and default prompt files if they don't exist.
func ensurePromptFiles(configDir string) error {
    promptsDir := filepath.Join(configDir, "prompts")

    // Create prompts directory if it doesn't exist
    if _, err := os.Stat(promptsDir); os.IsNotExist(err) {
        if err := os.MkdirAll(promptsDir, 0o755); err != nil {
            return fmt.Errorf("failed to create prompts directory: %w", err)
        }
    }

    // Default prompt file contents
    defaults := map[string]string{
        "commit-message.txt": getDefaultCommitPrompt(),
        "code-review.txt":    getDefaultCodeReviewPrompt(),
        "style-review.txt":   getDefaultStyleReviewPrompt(),
    }

    for filename, content := range defaults {
        path := filepath.Join(promptsDir, filename)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
                return fmt.Errorf("failed to write default prompt file %s: %w", filename, err)
            }
        }
    }

    return nil
}

// getDefaultCommitPrompt returns the default commit message prompt template.
func getDefaultCommitPrompt() string {
    return `You are an expert software engineer.
Analyze the provided git diff and generate a commit message following the Conventional Commits specification.

### 1. INPUT
- **Diff Data**: See below.
- **Target Language**: {LANGUAGE}

### 2. RULES
- **Intent**: Focus on *why* the change was made.
- **Noise**: Ignore pure formatting changes or lockfiles unless specific intent exists.
- **Types**: Keep standard types (feat, fix, chore, etc.) in English.

### 3. OUTPUT FORMAT (Strictly Follow)
You must generate the message in the structure below, using **{LANGUAGE}** for the description and body:

<type>(<scope>): <concise description in {LANGUAGE}>

[Optional Body in {LANGUAGE}, bullet points]

### 4. EXAMPLES (For format reference only - Translate your output to {LANGUAGE})

Input: (Logic change)
Output: fix(auth): resolve nil pointer in token validation
*(If Language is Chinese, you should output: fix(auth): 修复令牌验证中的空指针问题)*

Input: (Ignore file)
Output: chore(gitignore): ignore .DS_Store files

---

### 5. FINAL INSTRUCTION
Analyze the diff below and write the commit message.
**CRITICAL**: The <description> and <body> MUST be in **{LANGUAGE}**.

### DIFF TO ANALYZE:
{DIFF}`
}

// getDefaultCodeReviewPrompt returns the default code review prompt template.
func getDefaultCodeReviewPrompt() string {
    return `Review the following code diff for potential issues, and provide suggestions, following these rules:
- Identify potential style issues, refactoring opportunities, and basic security risks if any.
- Focus on code quality and best practices.
- Provide concise suggestions in bullet points, prefixed with "- ".
- Be direct and avoid extraneous conversational text.
- Assume the perspective of a code reviewer offering constructive feedback to a developer.
- If no issues are found, explicitly state "No issues found."
- Language of the response MUST be {LANGUAGE}.

Diff:
{DIFF}
`
}

// getDefaultStyleReviewPrompt returns the default commit style review prompt template.
func getDefaultStyleReviewPrompt() string {
    return `Review the following commit message for clarity, informativeness, and adherence to best practices. Provide feedback in bullet points if the message is lacking in any way. Focus on these aspects:

- **Clarity**: Is the message clear and easy to understand? Would someone unfamiliar with the changes easily grasp the intent?
- **Informativeness**: Does the message provide sufficient context about *what* and *why* the change is being made? Does it go beyond just *how* the code was changed?
- **Diff Reflection**: Does the commit message accurately and adequately reflect the changes present in the Git diff? Is it more than just a superficial description?
- **Semantic Feedback**: If the message is vague or superficial, provide specific, actionable feedback to improve it (e.g., "This message is too vague; specify *why* this change is necessary", "Explain the impact of this change on the user").

If the commit message is well-written and meets these criteria, respond with the phrase: "No issues found."

Commit Message to Review:
{COMMIT_MESSAGE}

Language for feedback MUST be {LANGUAGE}.
`
}
