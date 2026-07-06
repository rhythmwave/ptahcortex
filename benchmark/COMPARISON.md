# Ptahcortex Benchmark: Model Comparison

## GLM-5.1 vs GLM-5.2

| Metric | GLM-5.1 | GLM-5.2 | Winner |
|--------|---------|---------|--------|
| Tokens | 14,353 | 11,831 | GLM-5.2 (-18%) |
| Duration | 2m 45s | 1m 31s | GLM-5.2 (-45%) |
| Quality | 7 findings | Partial | GLM-5.1 |

## Recommendation

- **GLM-5.2** → Quick searches, cost-sensitive
- **GLM-5.1** → Code reviews, quality-critical

## Token Efficiency (Context Manager)

With Context Manager: ~27K tokens
Without (estimated): ~35K tokens
**Savings: ~22%**
