# Explicit model roles and strict child results

Harvest source reference is commit27f0e9fb87c9d11afa27dc273ec5f85f09b1d111;
research/harvest/ASSESSMENT.md contains pinned implementation/test links and
maturity/license boundaries. No Harvest runtime was executed. This original Go
subset was independently implemented and tested against local scripted models;
it is not a Harvest compatibility layer or a claim of universal feature novelty.

## Model roles

Config.model_roles maps compaction/task/explore to existing named profiles.
There are no implicit model catalogs, role aliases, fuzzy choices, fallback
chains or automatic cost assumptions. Unknown roles/profiles fail configuration.
Unset roles inherit the active model. Explore falls back to task when not set;
explicit custom-agent profiles take precedence. Profile changes affect only
provider/model/context/output settings, never tools, grants, agent mode, lineage
or session storage. Parent plan still caps child mode, even with a build profile.
Existing environment overrides remain authoritative over selected profile fields;
root CLI overrides apply to root configuration, while explicitly routed operations
use their selected profile/environment settings.

Compaction routes through its own selected model, records model/profile/protocol
identity in the durable model_route and context_compacted metadata, and admits
input against that model's context/output budget before requesting a summary.
An insufficient summary model fails without changing history; no escalation to
another model is implicit. Child model/profile/protocol/context are shown in task
approval and run metadata. No actual pricing/cost or universal speed gain follows
from selecting a different configured model.

## Typed child output

Optional task.output_schema accepts a JSON Schema object (<=16KiB). Preflight
rejects invalid schemas, references/identifiers and excessive depth/nodes without
network/file loading, before a child session/model is created. Schemas are data,
not permission grants. The child is asked to return one plain JSON final value;
completed output is parsed with integer fidelity, duplicate-key/trailing-text
refusal and bounded depth/nodes. The serialized value is capped24KiB.

A valid result includes structured_output, output_schema_checked=true,
child_status=completed and normal child identity/summary. Invalid final JSON/schema
returns task status invalid_output with Error=true while retaining
child_status=completed, the task ID, bounded raw summary and an explanation.
Already completed effects are not rolled back or rerun; no automatic repair
model call occurs. Failed/interrupted/uncertain children retain their existing
status/recovery paths and never advertise a validated result. Validation is
per invocation, not an inherited policy silently attached to resumed child IDs.
The schema is retained in the child's subagent_request metadata for review.

No JTD conversion, schema inheritance, permissive coercion, incremental yield,
provider-native strict output mode or remote schema dereferencing is implemented.
The strict object-schema behavior and no-retry policy deliberately differ from
Harvest's broader caller/agent/session precedence and permissive modes.

## Verification

Original unit tests verify role precedence/custom overrides/plan caps, mixed
protocol credentials, summary input admission/no-history-loss, preflight no-fetch,
integer fidelity, valid/invalid/fenced/oversized/duplicate JSON, child effects
executed once, retained child IDs and no uncertain-start fabrication on schema
failure. The typed-child PTY scenario checks actual separate task approval and
durable structured output. Actual Go REA captures for valid/invalid tasks and
compaction are indexed in research/evidence/harvest-batch-driver.json. They use
controlled loopback endpoints only; selected assertions are not canonical REA
equivalence and bounded process-observation limits remain in full Evidence.
