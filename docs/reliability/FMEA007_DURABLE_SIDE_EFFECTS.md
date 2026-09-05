# FMEA-007 — durable side-effect contract

Status: closure candidate

## Invariant

A workflow retry/restart may re-execute control code, but it must never create a second externally observable effect for the same logical operation.

Logical operation identity is:

```text
(run_id, activity/node, iteration, effect kind)
```

The semantic payload has a separate digest. The payload-bound operation ID is derived from the logical identity plus that digest.

Therefore:

- same logical identity + same payload -> return the original receipt with `reused=true`;
- same logical identity + different payload -> fail with operation conflict;
- a new source operation may apply only when the current source digest equals the exact expected revision;
- source state + receipt are persisted in one atomic fsync+rename snapshot;
- memory admission uses the same logical receipt rule and independently deduplicates semantic graph edges/conflict links;
- attempts are never part of logical effect identity.

## Retry classes

| Class | Meaning | Examples |
|---|---|---|
| `safe_retry` | no externally visible mutation | impact analysis, critique, verification |
| `retry_with_idempotency_key` | target must recognize stable logical operation | source repair commit, memory admission |
| `non_retryable_human_review` | automatic replay cannot prove safe state transition | CAS conflict, logical-key payload drift, ambiguous external state |

## Axiom boundary

`iterate_repair` is declared `ExternalEffect=true`, has bounded retry and explicit `{execution}:{node}` Axiom idempotency identity. `ActivityRequest` identity is exposed to adapters through `ActivityIdentityFromContext`.

Per-iteration target operation IDs are finer-grained than the Axiom activity key because several repair iterations can execute inside one durable activity. This prevents a retry of the outer activity from duplicating already-completed iteration effects.

## Source commit

`sideeffect.SourceStore.CompareAndSwap` requires:

1. stable operation identity;
2. payload digest equal to desired source-state digest;
3. exact expected current source revision for a new operation.

The file-backed store persists desired state and the effect receipt atomically. Reopening after a simulated crash therefore reconstructs the receipt and makes replay a no-op.

## Memory admission

`EpMemoryStore.CommitOnce` records one receipt per logical admission operation. Legacy `Commit` also deduplicates exact edge/conflict identities so old callers cannot multiply graph structure on replay.

`HostRepairEngine` no longer swallows mapper/commit errors. A final PASS followed by failed memory admission is an execution failure, not a falsely successful complete workflow.

## Fault evidence required for closure

- source effect persisted, process/store reopened before activity completion, same operation replays as `reused=true`;
- concurrent source revision produces CAS conflict;
- same logical operation with mutated payload fails closed;
- memory retry creates exactly one atom/edge set;
- legacy repeated commit does not duplicate conflict edges;
- Axiom external-effect activity deliberately fails transiently after source + memory effects, retries under the same activity idempotency key, and both target effects report reuse;
- durable Axiom history exposes the same `idempotencyKey` on the failed and retried attempts;
- reopening a completed file-backed Axiom run loads history/result without replaying external effects.

Residual target after all gates pass: `S=9 O=1 D=2 RPN=18`.
