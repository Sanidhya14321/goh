# Skill discovery and loading: pinned observable contract

Reference: OpenCode v1.19.5 at `810c0eba1a7348036f8bdf61e6632d0c36c6876c`. Artifact identity is retained in research/reference-manifest.json. The executable's active legacy skill implementation is distinct from the newer core/v2 source branch; these observations target the actual pinned headless CLI, not an assumption that both branches behave identically.

## Evidence and isolation

All fixtures are independently authored under research/skills. The disposable project is a Git repository, establishing a traversal boundary. HOME and every XDG directory point to controlled experiment folders. No user home skill directory was scanned or changed. Only the local scripted endpoint was contacted; no remote skill URL was configured.

| Record | Evidence ID | Observation |
|---|---|---|
| reference-skills.json | ev_7ccb6155ccec09949bfb491f0b454c3758fcca1442ef5c6d27540ce71690b029 | Eight load attempts covering roots, frontmatter, duplicates, lazy body and missing names. Exit0. Rendered terminal budget truncated; listed raw tool results retained. |
| reference-skills-paths.json | ev_186b62e7f5b145651aa66cb1c3b84990d8d930d6f7543f3e165906c1513c30d2 | Controlled .claude root, isolated home .agents root and explicit skills.paths all load. Exit0; rendered budget truncated. |
| reference-skills-ask.json | ev_97ebb3c851878debb793b77b99f22a0a6c89314dc28b929ba664a6a0a54df4fa | Name-specific ask auto-rejected headlessly; tool error, exit0. |
| reference-skills-deny.json | ev_91a71f535fc5feedfc75a74ea700ac331345f9e7b3284b646500c8ad3f5f32be | Name-specific deny hides metadata but a forced call yields permission error; exit0. Rendered budget truncated. |

Model request logs are companion fixture observations, not separately canonical REA Evidence. They contain reference-owned prompts and stay exclusively in research. The independently generated metadata projections omit upstream prompt text.

## Discovery and frontmatter

Observed roots:

- Project `.agents/skills/**/SKILL.md` and `.claude/skills/**/SKILL.md`.
- Project `.opencode/skills/**/SKILL.md` and singular `.opencode/skill/**/SKILL.md`.
- Controlled HOME `.agents/skills/**/SKILL.md`.
- Explicit configured `skills.paths` recursively containing SKILL.md.

Nested directories are accepted. The frontmatter `name` does not have to match the directory. The name `Fixture Upper Name!` loaded successfully, so the active CLI does not restrict names to lowercase slug syntax. `name` must be a string; a missing name or numeric name did not enter the catalog. `description` is optional, but if present must be a string (source-assisted rule; optional omission observed). A skill without description was directly loadable by name but omitted from initial advertised metadata.

An unquoted YAML value containing a colon and space, `description: Fixture: unquoted colon value`, parsed and advertised successfully. Source inspection explains that frontmatter uses gray-matter and retries sanitization for unquoted colon values. This bounded observation does not establish full YAML parser compatibility for arbitrary constructs. Unknown metadata, full block scalar/anchors/alias behavior and malformed-frontmatter error events need independent cases.

Two same-name fixture skills were found in .agents and .opencode; .opencode content won the observed run. Source load operations are concurrent and overwrite a shared name map, so this is not evidence of a guaranteed stable precedence rule. Do not present one timing-sensitive outcome as universal order. A deterministic Go order must be documented as a deliberate policy until stronger reference precedence is established.

## Advertisement and loading

The tool schema is `{name:string}` with the name required. No other field is required. Initial system context advertised described skill names, descriptions and locations sorted by name. Skill bodies and resource contents were absent before loading. The tool's ordinary description did not embed all bodies.

A successful `skill` call returns the selected trimmed body, base directory, instructions that relative resource paths resolve from that directory, and a sampled resource file list. The fixture resource filename appeared; its sentinel content did not. Body loading is lazy relative to model context, while source discovery may parse/cache files earlier. Do not describe this as filesystem-on-demand or hot reload without further evidence.

Tool result metadata contains the selected name and directory; title identifies the loaded skill. A missing name returns an error listing available skill names. The list includes directly loadable undescribed skills and the built-in customization skill. The built-in reference prompt body must not be copied into original Go material.

Source assistance identifies a maximum ten sampled resource files, SKILL.md exclusions, ripgrep-based sampling and no symlink following in the resource listing. The fixture observed one resource; maximum-limit and symlink behavior remain source-only, not runtime verified.

## Authority

With the controlled config's permission key absent, skill loads succeeded. A name-specific ask rule produced a headless permission rejection, and a deny rule produced an explicit policy error. The name denied in one fixture was absent from advertised initial metadata; other allowed fixture names remained. A model forced to call it still received permission denial, so hiding metadata is not the sole enforcement boundary.

Loading a skill supplies task guidance. It never grants file mutation, shell, MCP or network authority. Skill text and resources are untrusted content; resource files should remain lazy and tool access governed by existing policy.

## Source-assisted differences and remaining gates

Pinned source examined normally (not mislabeled REA static evidence): packages/opencode/src/skill/index.ts, tool/skill.ts, core/src/skill.ts, skill/discovery.ts, tool/skill.ts, plugin/skill.ts and config/markdown.ts. The active legacy CLI requires explicit frontmatter names. The separate v2 source supports other root .md naming/slash cases; those were not observed and must not silently replace the pinned active behavior contract.

Source-only surfaces still to investigate: config-directory order, global .claude root, external-skill disabling flags, remote index URLs/version refresh, duplicate races, cache invalidation, broken symlinks, invalid YAML, unsupported description types, resource limits and special characters in output XML. No remote installation or global scanning was authorized or performed here.

Original Go implementation and paired capture: pending. Acceptance should test discovery roots, metadata-only initial context, body/resource laziness, missing/invalid names, undescribed direct loads, colon frontmatter, denied/asked loads, explicit bounds, project containment policy and honest duplicate precedence. A stricter safe path/frontmatter policy is acceptable only as a documented intentional difference.

## Original Go implementation and paired observation

Implemented under `internal/skills` and shared runtime `skill`
admission. Project roots are .agents/skills, .claude/skills, .opencode/skills,
.opencode/skill, then original .harness/skills; explicitly configured skill_paths
are scanned afterward. No implicit home/ancestor scan or remote index fetch.
Deterministic first-discovered definition wins; duplicate/invalid files produce
bounded diagnostics. This differs from reference concurrent overwrite behavior.

A deliberately bounded frontmatter subset supports string scalars, quoted values,
plain colon-containing values and block descriptions. Arbitrary YAML aliases,
tags and full gray-matter compatibility are not claimed. Nonempty names cannot
contain control characters or path separators. Name/description/body/discovery
budgets and no-symlink/special-file rules are in README. Undescribed skills are
loadable by name and omitted from advertised metadata. Metadata lives in the Go
tool description rather than duplicating the reference's system prompt format.

Every load requires explicit named `skill` approval; it checks the discovery hash
before returning body/base/resource names as JSON. Resource contents remain lazy;
no scripts execute and no permissions are granted by the loaded text. External
configured resource files remain outside ordinary project file-tool scope. TUI
/skills lists the catalog locally; this is an original UI extension.

Go REA observations: go-skills.json (`ev_f225b78e40c745feac6d5aa4d9f803a7fb8c7f9305bd5eb2a238aa8e551f1020`),
go-skills-ask.json (`ev_dc5a36e2493158ef8d65ef9ceff4d46a34306f04f05236b691ad0f032fe78733`),
go-skills-plan.json (`ev_774e8d1ab87f0104043d3fb272c1c947b99f8c462e1f016f1bf73cc1a8a19332`).
Seven declared existing skill names load, the missing name errors, headless
ungranted load stops needs_permission without a start, and plan permits an
explicitly granted skill read. Original semantic checks retain these facts in
rest-semantic-comparisons.json; they do not claim full root/YAML/authority parity.

Tests cover discovery roots, metadata/body/resource separation, unknown/optional
metadata, explicit roots, duplicate precedence, malformed types, symlink/FIFO
refusal, changed files, resource caps, cancellation and runtime admission.
Bounded parser fuzzing ran 270,248 executions without failure. References remain
separate from original Go code/prompts.
