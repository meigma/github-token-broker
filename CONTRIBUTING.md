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

Branches are integrated through squash-merge, so commit titles within a branch do not need to follow a specific convention — the merged commit title is the PR title. Keep PR titles imperative and concise.

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

### Smoke test

`broker:smoke` runs an integration test that exercises the main binary against in-process stubs for the AWS Lambda Runtime API, AWS SSM, and the GitHub App endpoint. It boots a host-native build of `cmd/github-token-broker`, invokes it once with an empty payload, and asserts the response body carries a freshly minted token for the configured target repository.

```sh
moon run broker:smoke
# or
just smoke
```

The smoke test is gated behind the `integration` build tag so it does not run during `broker:test`. `moon ci` runs it automatically alongside the unit suite.

### Docs site

```sh
moon run docs:build
```

## License

This project is dual-licensed under [Apache License 2.0](LICENSE-APACHE) and the [MIT License](LICENSE-MIT). By submitting a contribution, you agree that your work may be distributed under either license, at the recipient's option. No separate CLA is required.
