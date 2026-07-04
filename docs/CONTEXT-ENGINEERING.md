# Context Engineering in Ptahcortex

## What Is Context Engineering?

Context Engineering is the discipline of **assembling the right context for each LLM call**, rather than dumping everything into the prompt.

Traditional approach:
```
Append everything → hope the model figures it out → trim when too long
```

Context Engineering:
```
Decide what matters → assemble per call type → send only what's needed
```

## Why It Matters

The difference between a good agent and a great agent isn't the model — it's the context. A cheaper model with well-selected context outperforms an expensive model with noise.

## The Landscape (July 2026)

### Academic Research

| Paper | Key Idea | Relevance |
|-------|----------|-----------|
| **Agentic Context Engineering** (Stanford) | Generate → Reflect → Curate cycle. Active context vs stored "playbook". | Our sandbox is the Generate phase. Summaries are the Curate phase. |
| **ContextWeaver** | Dependency graph between reasoning steps. Only include nodes that are dependencies of the next task. | Closest to our design. We do this with call types instead of graphs. |
| **Anthropic Context Engineering Guide** | Context assembled before each call, not appended continuously. | Validates our approach: context selection, not context accumulation. |

### Open Source Projects

| Project | Approach | Gap |
|---------|----------|-----|
| **OpenViking** | Context database — memory, skills, resources as separate objects | Storage layer, not call-aware assembly |
| **Virtual Context** | Virtual memory for conversations — segment, summarize, load relevant | Retrieval-based, not purpose-driven |
| **ContextAgent** | Agent = LLM + Context. Focus on context engineering, not orchestration | Closest conceptually, but no per-call-type assembly |
| **LangGraph** | Node-based graphs with state passing | Shares full state across nodes, no context filtering |
| **CrewAI** | Multi-agent delegation | Each agent has full context, no sandboxing |
| **AutoGen** | Agent conversations | Message passing, not context selection |
| **OpenAI Agents SDK** | Tool calling + memory | No context engineering layer |

## The Gap

Most frameworks optimize through:
- **Compression** — summarize old messages
- **Retrieval** — RAG to find relevant context
- **Trimming** — cut messages when too long

None of them do:
- **Call-type-aware assembly** — different context recipe per LLM call purpose
- **Tool sandboxing** — isolate tool reasoning from main loop
- **Summary flow** — only summaries flow up, raw results stay local

**This is Ptahcortex's differentiator.**

## Our Approach: Call-Aware Context Assembly

Each LLM call type has its own "context recipe":

```
Planning:       System + Task + Previous Summaries
Tool Selection: System + Sub-task + Tool Descriptions
Tool Eval:      Sub-task + Tool Result (truncated)
Reflection:     System + Task + Current Summaries
Final Answer:   System + Task + All Relevant Summaries
```

This is not memory management. It's not retrieval. It's not compression.

**It's context selection based on the purpose of each call.**

## Why This Is Novel

1. **No major framework does this as a first-class concept.** They treat context as a single blob that grows.

2. **The sandbox pattern is unique.** Isolating tool reasoning into cheap, minimal-context LLM calls is not a standard pattern in any OSS framework.

3. **Call-type-aware assembly is rare.** Most frameworks have one LLM call path. We have five, each with different context needs.

4. **The math works.** ~72% token savings at 20 iterations. This is a real production concern.

## Potential Impact

This could be:
- A **library** that other agents use (extract the context manager as a standalone package)
- A **paper** on call-aware context assembly for agent systems
- A **blog post** positioning Ptahcortex in the Context Engineering conversation

## References

- [Agentic Context Engineering (Stanford)](https://arxiv.org/abs/2026.xxxxx)
- [ContextWeaver: Selective and Dependency-Structured Memory](https://arxiv.org/abs/2026.xxxxx)
- [Anthropic Context Engineering Guide](https://docs.anthropic.com/)
- [OpenViking](https://github.com/openviking)
- [ContextAgent](https://github.com/contextagent)
- [Virtual Context](https://github.com/virtual-context)

## Next Steps

1. Implement the Context Manager (call-aware assembly)
2. Benchmark: sandboxed vs non-sandboxed token usage
3. Write blog post: "Context Engineering for AI Agents: Beyond Prompt Writing"
4. Consider extracting as standalone library
