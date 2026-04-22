# github-token-broker

`github-token-broker` is a small AWS Lambda function that vends short-lived, scoped GitHub App installation tokens.
It is intended for bootstrap and automation workflows that need a GitHub token but should not carry long-lived credentials themselves, and is maintained by the [meigma](https://github.com/meigma) organization.

> **Status:** this repository is being bootstrapped from an internal service. The Go implementation will land in a follow-up change; until then, only the repository scaffolding is in place.

## How it works

The Lambda reads three values from AWS SSM Parameter Store — a GitHub App client ID, an installation ID, and the App's RSA private key — signs a short-lived JWT, and exchanges it with the GitHub API for an installation token scoped to the configured repository and permissions. Callers receive the token and its expiration and use it for the lifetime of their task.

Boundaries kept deliberately small:

- No secrets are stored outside AWS SSM; the broker only reads them to mint a token.
- The Lambda accepts empty input and returns a JSON payload; it is not an open HTTP API.
- The issued token is always scoped — a compromise is bounded to the target repository and permissions you configure.

## Prerequisites

To run your own broker you will need:

- A GitHub App registered in your organization or account, with an RSA private key (PEM) and at least one installation.
- An AWS account with permission to deploy a Lambda function and write SSM parameters.
- Go (matching the version in `go.mod` once it lands) and the Moon toolchain used by this repo.

## Configuration

The broker reads configuration from environment variables and SSM:

- `AWS_REGION` — AWS region for SSM and Lambda. Required.
- SSM parameter names for `client_id`, `installation_id`, and `private_key`. Defaults and overrides will be documented alongside the implementation.
- Target repository and permissions for the issued token.

Exact parameter names, defaults, and the response schema will be documented in `docs/` as the implementation lands.

## Documentation

The Docusaurus site under [`docs/`](docs/) is the canonical location for configuration, deployment, and operational guidance. Published versions will be linked here once the site is deployed.

## Support

- Questions and general discussion: [GitHub Discussions](https://github.com/meigma/github-token-broker/discussions).
- Bug reports: [GitHub Issues](https://github.com/meigma/github-token-broker/issues).
- Do **not** report vulnerabilities in public channels. See [SECURITY.md](SECURITY.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines and pull request expectations.

## Security

See [SECURITY.md](SECURITY.md) for the private vulnerability reporting path.

## License

`github-token-broker` is dual-licensed under the [Apache License 2.0](LICENSE-APACHE) or the [MIT License](LICENSE-MIT), at your option.

Unless you explicitly state otherwise, any contribution intentionally submitted for inclusion in this project shall be dual-licensed as above, without any additional terms or conditions.
