package agentsession

import (
	"encoding/json"
	"strings"

	"latere.ai/x/pkg/ndjson"
)

// RoundUsage captures the token and cost fields from a single agent-session round's
// stream-json output. It mirrors the shape of agent usage reporting without
// pulling in the internal/store package, so the runtime stays free of a
// persistence dependency.
type RoundUsage struct {
	InputTokens              int
	OutputTokens             int
	CacheReadInputTokens     int
	CacheCreationInputTokens int
	CostUSD                  float64
	StopReason               string
}

// resultLine is the subset of the agent stream-json "result" message that the
// runtime needs to record per-round usage. The field layout matches the
// agentOutput shape used by internal/runner, but we decode locally to avoid
// pulling runner-side types into the runtime.
type resultLine struct {
	Type         string  `json:"type"`
	StopReason   string  `json:"stop_reason"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	Usage        struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
}

func (r resultLine) toRoundUsage() RoundUsage {
	return RoundUsage{
		InputTokens:              r.Usage.InputTokens,
		OutputTokens:             r.Usage.OutputTokens,
		CacheReadInputTokens:     r.Usage.CacheReadInputTokens,
		CacheCreationInputTokens: r.Usage.CacheCreationInputTokens,
		CostUSD:                  r.TotalCostUSD,
		StopReason:               r.StopReason,
	}
}

// ExtractUsage scans NDJSON output for the final "result" line and returns
// its token and cost fields. It prefers a line with a non-empty stop_reason
// (the terminal result message emitted at the end of a round) and falls
// back to the last well-formed result object if none is present.
//
// Returns ok=false when no usable result line is found.
func ExtractUsage(raw []byte) (RoundUsage, bool) {
	// Claude/Codex-style terminal result frame.
	obj, ok := ndjson.PreferResultLine(string(raw), true,
		func(r *resultLine) bool { return r.Type == "result" || r.Type == "" },
		func(r *resultLine) bool { return r.StopReason != "" })
	if ok {
		return obj.toRoundUsage(), true
	}

	// Pi keeps usage on assistant messages. Prefer agent_end (which carries the
	// completed conversation), then message_end as a fallback.
	type piMessage struct {
		Role       string `json:"role"`
		StopReason string `json:"stopReason"`
		Usage      *struct {
			Input      int `json:"input"`
			Output     int `json:"output"`
			CacheRead  int `json:"cacheRead"`
			CacheWrite int `json:"cacheWrite"`
		} `json:"usage"`
	}
	type piLine struct {
		Type     string      `json:"type"`
		Message  *piMessage  `json:"message"`
		Messages []piMessage `json:"messages"`
	}
	toUsage := func(msg *piMessage) (RoundUsage, bool) {
		if msg == nil || msg.Role != "assistant" || msg.Usage == nil {
			return RoundUsage{}, false
		}
		return RoundUsage{
			InputTokens:              msg.Usage.Input,
			OutputTokens:             msg.Usage.Output,
			CacheReadInputTokens:     msg.Usage.CacheRead,
			CacheCreationInputTokens: msg.Usage.CacheWrite,
			StopReason:               msg.StopReason,
		}, true
	}

	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || line[0] != '{' {
			continue
		}
		var pi piLine
		if json.Unmarshal([]byte(line), &pi) != nil {
			continue
		}
		switch pi.Type {
		case "agent_end":
			for j := len(pi.Messages) - 1; j >= 0; j-- {
				if u, ok := toUsage(&pi.Messages[j]); ok {
					return u, true
				}
			}
		case "message_end":
			if u, ok := toUsage(pi.Message); ok {
				return u, true
			}
		}
	}
	return RoundUsage{}, false
}
