# Model profiles and primary agents

## Reference evidence

The pinned OpenCode v1.19.5 reference source declares built-in primary `build` and `plan` agents, plus subagents and hidden internal agents. Source assistance is saved in `research/reference/features/agent.ts` and `agent-cli.ts`; the previously saved run command source contains selection/fallback behavior. The source describes build permission-driven operation, plan edit restrictions with plan-document exceptions, merged user permission overrides, agent prompts and optional models. These are source facts, separate from runtime observations.

Actual REA process captures:

| Capture | Evidence ID | Observation |
|---|---|---|
| `agent-reference-list.json` | `ev_eb2175d85c334fcdc1970aebc6723cf60783738a2743033fcfed7c4925d98976` | `agent list` exits0 and prints named agents, primary/subagent modes and permission rules. Output includes discovered host skill path rules; HOME/XDG isolation does not establish a sandbox or complete host-discovery isolation. |
| `agent-reference-build.json` | `ev_c24449fd1dc0a161c7b1cd3b2aa9245a8ef911f666760dca7a503003d1126f36` | Explicit `run --agent build --model fixture/scripted` completes a text turn using only the local model fixture. |
| `agent-reference-plan.json` | `ev_d249d1e7ba9b935693eac686213066459a2efead482a165bb21b377fadfac2b2` | Explicit `run --agent plan --model fixture/scripted` completes the same bounded text turn, preserving file snapshots. |

The latter two captures establish selection and text operation, not enforcement against forced edits/shell in the reference. The scripted-model request ledger is retained in `session-fork-model-requests.jsonl`; no paid provider or live credentials were used. Reference custom-agent execution, subscription/provider-specific behavior and all model-family capabilities are unverified here.

## Original implementation contract

User JSON configuration can declare `profiles`, `agents`, and `skill_paths`. A model profile may override endpoint, model, context window, output reserve, output-token parameter and API credential **environment-variable name**. Raw API keys are unsupported fields. Agent profiles contain description, independently supplied instructions, mode `build` or `plan`, and an optional model-profile preference. Built-in `build` and `plan` cannot be replaced. Custom agent instructions never create grants.

```json
{
  "profile": "local",
  "agent": "build",
  "profiles": {
    "local": {
      "endpoint": "http://127.0.0.1:8080/v1/chat/completions",
      "model": "explicit-model-name",
      "context_window": 64000,
      "output_budget": 4096,
      "api_key_env": "LOCAL_MODEL_KEY"
    }
  },
  "agents": {
    "reviewer": {
      "description": "Review the project",
      "instructions": "Explain findings with file evidence.",
      "mode": "plan",
      "profile": "local"
    }
  },
  "skill_paths": ["/explicit/user/skills"]
}
```

`harness run|chat|resume --profile NAME --agent NAME` selects a profile and primary agent. Unknown names fail clearly rather than falling back silently. An explicitly selected/configured model profile wins over the custom agent's optional profile preference. Selection precedence is explicit CLI selector, supported HARNESS_PROFILE/HARNESS_AGENT environment selector, configuration selector, then default `build`. Provider field precedence is defaults → base config → selected profile → supported environment variables → explicit CLI flags. Existing HARNESS_ENDPOINT, HARNESS_MODEL, HARNESS_CONTEXT_WINDOW, HARNESS_API_KEY_ENV, HARNESS_SESSION_DIR behavior is retained; HARNESS_OUTPUT_BUDGET is also supported. Numeric profile overrides use pointers so invalid explicit zeros reach validation rather than being ignored. Runtime-only AgentMode, AgentInstructions and AgentDescription are excluded from JSON serialization.

`plan` is deliberately stricter than the reference: no project writes, shell or executable MCP integrations regardless of grants or custom prompt. Read tools and explicitly admitted non-executable instruction resources remain usable. Runtime enforcement and event/UI display are shared-runtime responsibilities, not a second loop in the configuration layer. No OpenCode agent config import, markdown agent creation wizard, subagent execution or provider subscription profile compatibility is claimed.

## Verification

Config tests cover selected agent/profile defaults, custom profile preference versus explicit selection, environment/CLI override precedence, invalid numeric overrides, unknown selections, reserved built-in policies, unsupported authority/raw-key fields, strict JSON trailing data and skill root propagation. CLI tests use a local httptest SSE service to verify the model actually requested through selected profile and explicit model override. Runtime policy tests and Go REA captures are retained in the coordinated implementation checkpoint.

## Shared-runtime enforcement and Go capture

Runtime policy now filters tool advertisement and rejects forced calls in
prepare/execute/permission paths. Plan can read project files, load an explicitly
approved skill and ask questions; no writes, task mutations, shell, patches or
MCP startup/calls execute even with run/session grants or contrary custom-agent
instructions. Current trusted system framing is restored on replay, and selected
agent/profile/mode appear in the TUI header, events and non-authorizing run records.
Mode/profile are per invocation; they are not restored as authority from history.

`go-plan-forced-write.json`, `ev_f208b771b7f0d7806c14dbb5cc29345e53452dcda6bf840ac02bad3412009223`,
uses a selected model profile and custom plan reviewer against a loopback fixture.
The forced create call returns a tool error, no tool starts and no file appears,
despite write/shell/MCP grants and contrary custom-agent text. Original runtime
tests also force shell/task/direct policy paths and verify no MCP factory startup.
This strict mode is intentionally narrower than OpenCode's observed/configurable
plan rules and plan-file writes; no full agent-profile parity is claimed.
