package context

// CallType identifies the kind of LLM call being made.
// Each call type gets a different context recipe (which tiers to include).
type CallType int

const (
	// CallPlan decides what to do next based on task + previous summaries.
	CallPlan CallType = iota

	// CallSandboxSelect chooses which tool to call with what arguments.
	// Minimal context: just tool definitions + sub-task.
	CallSandboxSelect

	// CallSandboxEval interprets a tool result and extracts key information.
	// Context: sub-task + truncated tool result.
	CallSandboxEval

	// CallReflect evaluates progress after sandbox calls complete.
	// Context: task + sandbox summaries + previous summaries.
	CallReflect

	// CallFinal produces the final answer for the user.
	// Context: task + all summaries.
	CallFinal
)

// String returns the human-readable name for a CallType.
func (ct CallType) String() string {
	switch ct {
	case CallPlan:
		return "plan"
	case CallSandboxSelect:
		return "sandbox_select"
	case CallSandboxEval:
		return "sandbox_eval"
	case CallReflect:
		return "reflect"
	case CallFinal:
		return "final"
	default:
		return "unknown"
	}
}

// ContextTier identifies a category of context content.
type ContextTier int

const (
	// TierSystem — system prompt + tool definitions. Always included.
	TierSystem ContextTier = iota

	// TierTask — original user task. Always included.
	TierTask

	// TierToolResults — current iteration's tool results. Included for eval/reflect.
	TierToolResults

	// TierSummaries — previous iterations' sandbox summaries. Included for plan/reflect/final.
	TierSummaries

	// TierFullHistory — full raw history. Never used (replaced by summaries).
	TierFullHistory
)

// SandboxResult holds the output of an isolated sandbox LLM call.
type SandboxResult struct {
	SubTask    string // the sub-task description
	ToolName   string // which tool was called
	Summary    string // LLM-generated summary of the tool result
	RawResult  string // original tool output (for debugging)
	TokensUsed int    // tokens consumed by this sandbox call
}

// ContextStats tracks token usage across call types.
type ContextStats struct {
	PlanTokens       int
	SandboxTokens    int
	ReflectTokens    int
	FinalTokens      int
	TotalTokens      int
	SandboxCallCount int
}

// AddRecord records tokens for a specific call type.
func (cs *ContextStats) AddRecord(ct CallType, tokens int) {
	switch ct {
	case CallPlan:
		cs.PlanTokens += tokens
	case CallSandboxSelect, CallSandboxEval:
		cs.SandboxTokens += tokens
	case CallReflect:
		cs.ReflectTokens += tokens
	case CallFinal:
		cs.FinalTokens += tokens
	}
	cs.TotalTokens += tokens
	if ct == CallSandboxSelect || ct == CallSandboxEval {
		cs.SandboxCallCount++
	}
}
