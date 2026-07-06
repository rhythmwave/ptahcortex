#!/bin/bash
# Benchmark runner for Ptahcortex
# Usage: ./run-benchmark.sh [model] [task_number]

set -e

MODEL="${1:-glm-5.1}"
TASK="${2:-all}"
RESULTS_DIR="$(dirname "$0")/results"
CONFIG_TEMPLATE="$(dirname "$0")/config-template.yaml"

mkdir -p "$RESULTS_DIR"

# Benchmark tasks
declare -A TASKS
TASKS[1]="List all Go files in the project"
TASKS[2]="Find all functions named Call or Start in the codebase"
TASKS[3]="Review error handling in the MCP client code"
TASKS[4]="Trace all callers of the CallTool function"
TASKS[5]="Find potential race conditions in concurrent code"

run_task() {
    local task_num=$1
    local task_desc="${TASKS[$task_num]}"
    local result_file="$RESULTS_DIR/${MODEL}_task${task_num}_$(date +%Y%m%d_%H%M%S).json"
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "TASK $task_num: $task_desc"
    echo "Model: $MODEL"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Create temp config
    local tmp_config="/tmp/benchmark_${MODEL}_task${task_num}.yaml"
    sed "s/{{MODEL}}/$MODEL/g" "$CONFIG_TEMPLATE" > "$tmp_config"
    
    # Run and capture output
    local start_time=$(date +%s%N)
    local output=$(ptahcortex --config "$tmp_config" --task "$task_desc" 2>&1)
    local exit_code=$?
    local end_time=$(date +%s%N)
    local duration_ms=$(( (end_time - start_time) / 1000000 ))
    
    # Extract metrics from output
    local total_tokens=$(echo "$output" | grep -oP 'total tokens: \K[0-9]+' | tail -1)
    local iterations=$(echo "$output" | grep -oP 'iteration=\K[0-9]+' | tail -1)
    local tool_calls=$(echo "$output" | grep -oP 'sandbox: [0-9]+ tokens \(\K[0-9]+' | tail -1)
    local plan_tokens=$(echo "$output" | grep -oP 'plan:    \K[0-9]+' | tail -1)
    local sandbox_tokens=$(echo "$output" | grep -oP 'sandbox: \K[0-9]+' | tail -1)
    local reflect_tokens=$(echo "$output" | grep -oP 'reflect: \K[0-9]+' | tail -1)
    local final_tokens=$(echo "$output" | grep -oP 'final:   \K[0-9]+' | tail -1)
    
    # Get final answer
    local final_answer=$(echo "$output" | sed -n '/RUN COMPLETE/,/═══/p' | head -n -1)
    
    # Create JSON result
    cat > "$result_file" << EOF
{
    "model": "$MODEL",
    "task_num": $task_num,
    "task": "$task_desc",
    "timestamp": "$(date -Iseconds)",
    "metrics": {
        "total_tokens": ${total_tokens:-0},
        "iterations": ${iterations:-0},
        "tool_calls": ${tool_calls:-0},
        "duration_ms": $duration_ms,
        "plan_tokens": ${plan_tokens:-0},
        "sandbox_tokens": ${sandbox_tokens:-0},
        "reflect_tokens": ${reflect_tokens:-0},
        "final_tokens": ${final_tokens:-0}
    },
    "exit_code": $exit_code
}
EOF
    
    echo ""
    echo "Results saved to: $result_file"
    echo ""
    echo "METRICS:"
    echo "  Total Tokens:    ${total_tokens:-N/A}"
    echo "  Iterations:      ${iterations:-N/A}"
    echo "  Tool Calls:      ${tool_calls:-N/A}"
    echo "  Duration:        ${duration_ms}ms"
    echo "  Plan Tokens:     ${plan_tokens:-N/A}"
    echo "  Sandbox Tokens:  ${sandbox_tokens:-N/A}"
    echo "  Reflect Tokens:  ${reflect_tokens:-N/A}"
    echo "  Final Tokens:    ${final_tokens:-N/A}"
    echo ""
}

# Main
if [ "$TASK" = "all" ]; then
    for i in 1 2 3 4 5; do
        run_task $i
        echo ""
    done
else
    run_task "$TASK"
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "BENCHMARK COMPLETE"
echo "Results in: $RESULTS_DIR"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
