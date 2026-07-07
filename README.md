# ssl-tournament-package

Organizational tooling for running a RoboCup SSL tournament — teams, format & brackets,
referee assignment, scores and standings. API-first: the web UI and any external tools are
clients of it.

Built in **Go** — a single self-contained binary with the web UI embedded, cross-compiled
for Linux/macOS/Windows (modeled on
[ssl-game-controller](https://github.com/RoboCup-SSL/ssl-game-controller)).

Early planning stage — see [docs/architecture.md](docs/architecture.md).

Sibling to `ssl-streaming-package` (the live-streaming tier).
