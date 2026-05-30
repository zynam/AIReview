package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"aireview/internal/review"
)

const systemPrompt = `你是一个 AI Pull Request 代码审查助手。
只能基于用户提供的 PR 元信息、patch 内容和可选辅助上下文进行审查。
只返回 JSON，不要使用 Markdown 代码块包裹，也不要输出额外说明。
每条 finding 都必须包含 file、line、evidence、suggestion、severity、confidence、category、title 和 needs_human_check。
如果证据不充分或结论存在不确定性，必须降低 confidence，并设置 needs_human_check=true。
rule_findings 只是辅助线索，不是最终结论；不要机械复述它们，必须结合 patch 独立判断后再输出 finding。`

func buildPrompt(req ReviewRequest) (string, error) {
	payload := promptPayload{
		Language:     req.Config.Language,
		ReviewFocus:  req.Config.ReviewFocus,
		PullRequest:  req.PullRequest,
		RuleFindings: req.RuleFindings,
		OutputSchema: map[string]any{
			"summary":         "string",
			"impact":          []string{"string"},
			"findings":        []string{"Finding objects"},
			"test_assessment": "string",
			"skipped_files":   []string{"string"},
		},
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("build LLM prompt payload: %w", err)
	}

	return strings.Join([]string{
		"请分析这个 GitHub Pull Request，并返回一个 ReviewReport JSON 对象。",
		"请尽量使用请求中指定的语言填写 summary、impact、finding title、evidence、suggestion 和 test_assessment 等自然语言字段。",
		"不要编造未提供的文件、行号、行为、测试结果或上下文；所有结论都必须能从提供的 PR 信息或 patch 中追溯到依据。",
		"如果 rule_findings 中的辅助线索确实成立，请用请求语言重新组织为自然、去重后的 finding；如果只是常规提示或缺少 diff 依据，不要输出。",
		string(body),
	}, "\n\n"), nil
}

type promptPayload struct {
	Language     string             `json:"language"`
	ReviewFocus  []string           `json:"review_focus,omitempty"`
	PullRequest  review.PullRequest `json:"pull_request"`
	RuleFindings []review.Finding   `json:"rule_findings"`
	OutputSchema map[string]any     `json:"output_schema"`
}
