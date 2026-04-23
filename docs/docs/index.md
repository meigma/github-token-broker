---
title: github-token-broker
slug: /
sidebar_position: 0
hide_table_of_contents: true
description: Short-lived, scoped GitHub App installation tokens from AWS Lambda.
---

# github-token-broker

`github-token-broker` is a small AWS Lambda that exchanges GitHub App credentials stored in AWS SSM Parameter Store for a short-lived, scoped installation token. Callers invoke the Lambda with an empty payload and receive a token bound to one repository with deploy-time-configured permissions.

## At a glance

- Runs on AWS Lambda, Go on `arm64`.
- Reads a GitHub App's client ID, installation ID, and private key PEM from three SSM parameters.
- Mints an installation token via the GitHub API. The token is never logged.
- Scope — one owner/repo and one set of permissions — is fixed at deploy time. Callers cannot widen it.
- Shipped with a first-party Terraform module; consumers pin it from git.

## Where to go next

- **[Tutorial](./tutorials/deploy-your-first-broker)** — Deploy your first broker. A guided walkthrough from zero to a working Lambda that returns a token.
- **[How-to guides](./how-to/rotate-github-app-private-key)** — Operational tasks: rotate the private key, change the target repository, use with GitHub Enterprise Server.
- **[Reference](./reference/environment-variables)** — Exact shapes for every surface: env vars, response JSON, IAM policy, SSM parameters, error messages.
- **[Explanation](./explanation/architecture)** — Why it's designed this way: architecture diagrams, the security model, why empty payloads are enforced, why permissions are deploy-time.

## Resources

- [Repository on GitHub](https://github.com/meigma/github-token-broker)
- [Security policy](https://github.com/meigma/github-token-broker/blob/master/SECURITY.md)
- [Release notes](https://github.com/meigma/github-token-broker/releases)
