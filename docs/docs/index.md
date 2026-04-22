---
title: github-token-broker
slug: /
description: Short-lived, scoped GitHub App installation tokens from AWS Lambda.
---

# github-token-broker

`github-token-broker` is a small AWS Lambda that exchanges a GitHub App's private key — stored in AWS SSM Parameter Store — for a short-lived, scoped installation token and returns it to the caller.

This site will cover:

- **Concepts** — what the broker is, why you might want one, and the trust model it assumes.
- **Deploy** — registering the GitHub App, populating SSM parameters, and deploying the Lambda.
- **Invoke** — the expected input, the response schema, and the lifetime of issued tokens.
- **Operate** — logging, rotation, and common failure modes.

The Go implementation is being migrated from an internal service; these pages will grow as that lands. In the meantime, the [README](https://github.com/meigma/github-token-broker/blob/master/README.md) captures the current state, and [GitHub Discussions](https://github.com/meigma/github-token-broker/discussions) is the right place for early questions.
