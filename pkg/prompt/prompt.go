package prompt

import (
	"fmt"
	"strings"

	gogitobj "github.com/go-git/go-git/v5/plumbing/object"
	"github.com/renatogalera/ai-commit/pkg/committypes"
)

// DefaultPromptTemplate is used if no template is configured for commit message generation.
const DefaultPromptTemplate = `You are an expert software engineer.
Analyze the provided ` + "`git diff`" + ` and generate a commit message following the **Conventional Commits** specification.

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

// DefaultCodeReviewPromptTemplate is used for code review prompts.
const DefaultCodeReviewPromptTemplate = `Review the following code diff for potential issues, and provide suggestions, following these rules:
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

// DefaultCommitStyleReviewPromptTemplate is used for reviewing commit message style.
const DefaultCommitStyleReviewPromptTemplate = `Review the following commit message for clarity, informativeness, and adherence to best practices. Provide feedback in bullet points if the message is lacking in any way. Focus on these aspects:

- **Clarity**: Is the message clear and easy to understand? Would someone unfamiliar with the changes easily grasp the intent?
- **Informativeness**: Does the message provide sufficient context about *what* and *why* the change is being made? Does it go beyond just *how* the code was changed?
- **Diff Reflection**: Does the commit message accurately and adequately reflect the changes present in the Git diff? Is it more than just a superficial description?
- **Semantic Feedback**: If the message is vague or superficial, provide specific, actionable feedback to improve it (e.g., "This message is too vague; specify *why* this change is necessary", "Explain the impact of this change on the user").

If the commit message is well-written and meets these criteria, respond with the phrase: "No issues found."

Commit Message to Review:
{COMMIT_MESSAGE}

Language for feedback MUST be {LANGUAGE}.
`

// Updated defaultCommitSummaryTemplate to include language placeholder.
const defaultCommitSummaryTemplate = `Summarize the following git commit in markdown format.
Write the summary in {LANGUAGE}.

Use "###" to denote section titles. Include:

Rule 1: don't send any text other than the request, don't say you're sending markdown or anything
Rule 2 send everything from General Summary and nothing else
Rule 3: Do not send similar text like "Here's a summary of the git commit in markdown format:"

### General Summary
- Main purpose or key changes

### Detailed Changes
- Any noteworthy details (e.g., new features, bug fixes, refactors)

### Impact and Considerations
- Overview of how it affects the codebase and any considerations.

Commit Information:
Author: {AUTHOR}
Date: {DATE}
Commit Message:
{COMMIT_MSG}

Diff:
{DIFF}
`

// BuildCommitSummaryPrompt constructs the prompt used to ask the AI for a commit summary.
// It replaces placeholders with actual commit information and the diff string.
func BuildCommitSummaryPrompt(commit *gogitobj.Commit, diffStr, customPromptTemplate, language string) string {
	templateUsed := defaultCommitSummaryTemplate
	if strings.TrimSpace(customPromptTemplate) != "" {
		templateUsed = customPromptTemplate
	}

	promptText := strings.ReplaceAll(templateUsed, "{AUTHOR}", commit.Author.Name)
	promptText = strings.ReplaceAll(promptText, "{DATE}", commit.Author.When.Format("Mon Jan 2 15:04:05 MST 2006"))
	promptText = strings.ReplaceAll(promptText, "{COMMIT_MSG}", commit.Message)
	promptText = strings.ReplaceAll(promptText, "{DIFF}", diffStr)
	promptText = strings.ReplaceAll(promptText, "{LANGUAGE}", language)

	return promptText
}

// BuildCommitPrompt builds the prompt for generating a commit message.
// It replaces placeholders with the provided diff, language, commit type, and any additional context.
func BuildCommitPrompt(diff, language, commitType, additionalText, promptTemplate string) string {
	finalTemplate := promptTemplate
	if finalTemplate == "" {
		finalTemplate = DefaultPromptTemplate
	}

	commitTypeHint := ""
	if commitType != "" && committypes.IsValidCommitType(commitType) {
		commitTypeHint = fmt.Sprintf("- Use the commit type '%s'.\n", commitType)
	}

	promptText := strings.ReplaceAll(finalTemplate, "{COMMIT_TYPE_HINT}", commitTypeHint)
	promptText = strings.ReplaceAll(promptText, "{LANGUAGE}", language)
	promptText = strings.ReplaceAll(promptText, "{DIFF}", diff)

	additionalContextStr := ""
	if additionalText != "" {
		additionalContextStr = "\n\n[Additional context provided by user]\n" + additionalText
	}
	promptText = strings.ReplaceAll(promptText, "{ADDITIONAL_CONTEXT}", additionalContextStr)

	return promptText
}

// BuildCodeReviewPrompt builds the prompt for a code review.
// It replaces placeholders with the provided diff and language.
func BuildCodeReviewPrompt(diff, language, promptTemplate string) string {
	finalTemplate := promptTemplate
	if finalTemplate == "" {
		finalTemplate = DefaultCodeReviewPromptTemplate
	}

	promptText := strings.ReplaceAll(finalTemplate, "{LANGUAGE}", language)
	promptText = strings.ReplaceAll(promptText, "{DIFF}", diff)

	return promptText
}

// BuildCommitStyleReviewPrompt builds the prompt for reviewing the style of a commit message.
// It replaces placeholders with the commit message and language.
func BuildCommitStyleReviewPrompt(commitMsg, language, promptTemplate string) string {
	finalTemplate := promptTemplate
	if finalTemplate == "" {
		finalTemplate = DefaultCommitStyleReviewPromptTemplate
	}

	promptText := strings.ReplaceAll(finalTemplate, "{LANGUAGE}", language)
	promptText = strings.ReplaceAll(promptText, "{COMMIT_MESSAGE}", commitMsg)

	return promptText
}

func ExtractSummaryAfterGeneral(aiOutput string) string {
	markers := []string{"### General Summary", "General Summary"}
	for _, marker := range markers {
		index := strings.Index(aiOutput, marker)
		if index != -1 {
			return aiOutput[index:]
		}
	}
	return aiOutput
}
