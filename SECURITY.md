# Security Policy

This document explains how to report vulnerabilities in `github-token-broker` privately.

## Supported Versions

Only the latest release on the default branch receives security fixes. Older versions are not patched. Pin a released version and update when a new release is published.

## Reporting a Vulnerability

Report vulnerabilities through [GitHub's private vulnerability reporting](https://github.com/meigma/github-token-broker/security/advisories/new) for this repository.

Do not use public GitHub issues, pull requests, discussions, or other public forums for vulnerability reports.

When reporting a vulnerability, include as much of the following as possible:

- affected version, commit, or deployment identifier
- a description of the issue and its security impact
- steps to reproduce or a minimal proof of concept
- relevant logs, screenshots, or traces
- suggested mitigations or fixes, if available

Because this service brokers GitHub App tokens, please flag findings that could lead to token theft, scope escalation, or credential exposure with high priority in your report.

## Disclosure

Fixes are coordinated through GitHub security advisories on this repository. Reporters will be credited in the advisory unless they request otherwise. No response-time guarantees are made beyond reasonable effort.

## Key Rotation

If a GitHub App private key is suspected compromised, rotate it following the [rotate-github-app-private-key how-to](docs/docs/how-to/rotate-github-app-private-key.md). Overwriting the SSM SecureString parameter is a no-downtime rotation — the Lambda picks up the new key on the next invocation.
