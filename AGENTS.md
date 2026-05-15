# Agent instructions

Workspace-level conventions (git workflow, voice, ops boundary) load globally via `~/.claude/CLAUDE.md` -> `agentic-os-kai/AGENTS.md`. Nothing repo-specific to override yet; this file exists so the symmetric trifecta (README / AGENTS / docs/FEATURES) is complete and grep-discoverable.

## See also

- [README.md](README.md) - human-facing intro.
- [docs/FEATURES.md](docs/FEATURES.md) - inventory of what ships today.
- [.coily/coily.yaml](.coily/coily.yaml) - allowlisted commands. Agents route through coily, not bare `make` / `uv` / `python` / `npm` / `cargo` / `dotnet`.

Cross-reference convention from [coilysiren/agentic-os-kai#313](https://github.com/coilysiren/agentic-os-kai/issues/313).
