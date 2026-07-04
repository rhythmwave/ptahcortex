package context

// CallType identifies the kind of LLM call being made.
type CallType int

const (
	// CallPlan decides what to do next based on task + previous summaries.
	CallPlan CallType = iota

	// CallSandboxSelect chooses which tool to call with what arguments.
	CallSandboxSelect

	// CallSandboxEval interprets a tool result and extracts key information.
	CallSandboxEval

	// CallReflect evaluates progress after sandbox calls complete.
	CallReflect

	// CallFinal produces the final answer for the user.
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
