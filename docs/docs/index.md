---
title: github-token-broker
slug: /
description: Short-lived, scoped GitHub App installation tokens from AWS Lambda.
---

# github-token-broker

`github-token-broker` is a small AWS Lambda that exchanges GitHub App credentials stored in AWS SSM Parameter Store for a short-lived, scoped installation token and returns it to the caller.

The implemented Lambda has a deliberately narrow contract:

- It accepts only empty or `null` invocation payloads.
- It reads the GitHub App client ID, installation ID, and private key PEM from configured SSM parameters.
- It validates the configured owner/repository against the configured installation before minting a token.
- It requests the configured repository permissions, defaulting to `{"contents":"read"}`.
- It returns `token`, `expires_at`, `repositories`, and `permissions` as JSON.

Start with the [README](https://github.com/meigma/github-token-broker/blob/master/README.md) for build commands, environment variables, SSM defaults, and the response schema. Use [GitHub Discussions](https://github.com/meigma/github-token-broker/discussions) for usage questions while the deeper deployment and operations pages are filled in.
