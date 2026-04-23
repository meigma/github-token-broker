# Contributing

Thank you for your interest in contributing to `github-token-broker`.
This guide covers questions, bug reports, feature requests, and pull requests.
For private vulnerability reporting, use [SECURITY.md](SECURITY.md) instead of public channels.

## Asking Questions

Use [GitHub Discussions](https://github.com/meigma/github-token-broker/discussions) for usage questions, troubleshooting, and general discussion.

## Reporting Bugs

File non-security bugs as [GitHub Issues](https://github.com/meigma/github-token-broker/issues). Include:

- version, commit, or deployment identifier
- steps to reproduce
- expected behavior
- actual behavior
- relevant logs or a minimal reproduction

If you are reporting a security issue, stop and follow [SECURITY.md](SECURITY.md) instead.

## Proposing Features

Open a GitHub Discussion before starting work on larger changes. Describe the problem, the proposed approach, and any compatibility or migration concerns. Small, self-contained improvements can go straight to a pull request.

## Pull Requests

1. Keep changes focused and scoped to a single problem.
2. Add or update tests when behavior changes.
3. Update documentation under `docs/` when user-facing behavior changes.
4. Write a clear PR description tying the change to the problem it solves.
5. Make sure CI passes before requesting review.

### Conventional Commits (PR titles)

Branches are integrated through squash-merge, so only the PR title reaches `master`. Release automation reads that commit to decide version bumps and populate the changelog, so **PR titles MUST follow [Conventional Commits](https://www.conventionalcommits.org/)**.

Supported types:

| Type | When to use | Appears in CHANGELOG |
| --- | --- | --- |
| `feat` | User-facing feature or addition | Yes, under "Features" |
| `fix` | User-facing bug fix | Yes, under "Bug Fixes" |
| `perf` | Performance improvement | Yes, under "Performance" |
| `revert` | Reverts a previous change | Yes, under "Reverts" |
| `docs` | Documentation-only change | Hidden |
| `chore` | Maintenance, dependencies, internal cleanup | Hidden |
| `build` | Build system or toolchain change | Hidden |
| `ci` | CI/workflow change | Hidden |
| `test` | Test-only change | Hidden |

Breaking changes: use `feat!:` (or `fix!:`) in the PR title, or include a `BREAKING CHANGE:` footer in the PR body. Either form bumps the major version (post-1.0) and is surfaced prominently in the changelog.

Scopes are optional and generally unnecessary for this single-package repo. Per-commit titles within a branch remain free-form — they are squashed away and never reach `master`. Keep PR titles imperative, concise, and in lowercase after the type prefix.

## Local Setup

This repository uses [Moon](https://moonrepo.dev) to orchestrate tasks, with a developer-facing [Justfile](https://just.systems) wrapper for common recipes.

### Prerequisites

- [Moon](https://moonrepo.dev/docs/install) — Moon provisions the Go toolchain pinned in `.moon/toolchains.yml` (currently Go 1.26.2).
- [Just](https://just.systems/) — optional but convenient for local work.

### Go service

The service lives at the repository root: module path `github.com/meigma/github-token-broker`, entry point `cmd/github-token-broker`.

Build, test, and lint the Lambda from a clean checkout:

```sh
# Moon — authoritative in CI (this is what `moon ci` runs)
moon run broker:check

# Justfile — same commands, direct shell invocation
just check
```

Individual verbs (`fmt`, `test`, `build`) are available on both runners. `build` produces a reproducible `linux/arm64` Lambda zip at `dist/github-token-broker.zip`.

### Integration tests

`broker:integration` runs integration tests that exercise the main binary against a Testcontainers-managed Moto SSM server, an in-process AWS Lambda Runtime API stub, and a purpose-built GitHub App endpoint stub. The suite boots a host-native build of `cmd/github-token-broker`, invokes it with empty payloads, and covers successful token minting plus important SSM, GitHub, and private-key failure paths.

```sh
moon run broker:integration
# or
just integration
```

The integration suite is gated behind the `integration` build tag so it does not run during `broker:test`. It requires Docker for Testcontainers. `moon ci` runs it automatically alongside the unit suite.

### Docs site

```sh
moon run docs:build
```

## License

This project is dual-licensed under [Apache License 2.0](LICENSE-APACHE) and the [MIT License](LICENSE-MIT). By submitting a contribution, you agree that your work may be distributed under either license, at the recipient's option. No separate CLA is required.
