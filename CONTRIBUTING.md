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

This repository uses [Moon](https://moonrepo.dev) to orchestrate tasks. The Go implementation of the broker will land in a follow-up change; until then, only the docs site can be built locally:

```sh
moon run docs:build
```

Once the implementation lands, its build, test, and lint commands will be documented here and runnable via `moon run`.

## License

This project is dual-licensed under [Apache License 2.0](LICENSE-APACHE) and the [MIT License](LICENSE-MIT). By submitting a contribution, you agree that your work may be distributed under either license, at the recipient's option. No separate CLA is required.
