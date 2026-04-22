# github-token-broker

`github-token-broker` is an AWS Lambda function that vends short-lived, scoped GitHub App installation tokens.
It is intended for bootstrap and automation workflows that need a GitHub token but should not carry long-lived credentials themselves, and is maintained by the [meigma](https://github.com/meigma) organization.

## How It Works

The Lambda reads three values from AWS SSM Parameter Store: a GitHub App client ID, an installation ID, and the App's RSA private key. It signs a short-lived JWT, validates that the configured owner/repository belongs to that installation, then exchanges the JWT with the GitHub API for an installation token scoped to the configured repository and permissions.

Boundaries kept deliberately small:

- No secrets are stored outside AWS SSM; the broker only reads them to mint a token.
- The Lambda accepts only empty or `null` invocation payloads, so callers cannot request custom token scope.
- The issued token is scoped to one configured repository and the configured permission set.
- The broker returns token metadata only; it does not clone repositories or decrypt repository contents.

## Build and Test

This repository uses [Moon](https://moonrepo.dev) for CI task orchestration and a [Justfile](https://just.systems/) for local convenience.

```sh
moon run broker:check
moon run broker:integration
```

Equivalent Just recipes are available:

```sh
just check
just integration
```

`broker:check` runs formatting, unit tests, and the Lambda build. `broker:integration` runs the Docker-backed integration suite against a Moto SSM server, a Lambda Runtime API stub, and a GitHub App endpoint stub.

## Runtime Configuration

The broker reads configuration from environment variables:

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `AWS_REGION` | yes | none | AWS region used by the SDK for SSM. |
| `GITHUB_TOKEN_BROKER_REPOSITORY_OWNER` | yes | none | GitHub owner for the repository token. |
| `GITHUB_TOKEN_BROKER_REPOSITORY_NAME` | yes | none | GitHub repository name for the token. |
| `GITHUB_TOKEN_BROKER_CLIENT_ID_PARAM` | no | `/github-token-broker/app/client-id` | SSM parameter containing the GitHub App client ID. |
| `GITHUB_TOKEN_BROKER_INSTALLATION_ID_PARAM` | no | `/github-token-broker/app/installation-id` | SSM parameter containing the GitHub App installation ID. |
| `GITHUB_TOKEN_BROKER_PRIVATE_KEY_PARAM` | no | `/github-token-broker/app/private-key-pem` | SSM SecureString parameter containing the GitHub App private key PEM. |
| `GITHUB_TOKEN_BROKER_PERMISSIONS` | no | `{"contents":"read"}` | JSON object of GitHub repository permissions to request. |
| `GITHUB_TOKEN_BROKER_GITHUB_API_BASE_URL` | no | `https://api.github.com` | GitHub API base URL. Override mainly for tests or GitHub Enterprise compatibility. |
| `GITHUB_TOKEN_BROKER_LOG_LEVEL` | no | `info` | One of `debug`, `info`, `warn`, or `error`. |

The SSM parameter names must be absolute paths. The private key parameter should be a SecureString.

## Invocation Contract

Invoke the Lambda with an empty payload or JSON `null`. Any other payload is rejected.

Successful responses have this shape:

```json
{
  "token": "ghs_...",
  "expires_at": "2026-04-22T00:00:00Z",
  "repositories": ["example-owner/example-repo"],
  "permissions": {
    "contents": "read"
  }
}
```

The token is intentionally logged nowhere. Callers should treat it as a secret and discard it after the task completes.

## Documentation

The Docusaurus site under [`docs/`](docs/) is the canonical location for configuration, deployment, and operational guidance. Published versions will be linked here once the site is deployed.

## Support

- Questions and general discussion: [GitHub Discussions](https://github.com/meigma/github-token-broker/discussions).
- Bug reports: [GitHub Issues](https://github.com/meigma/github-token-broker/issues).
- Do not report vulnerabilities in public channels. See [SECURITY.md](SECURITY.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines and pull request expectations.

## Security

See [SECURITY.md](SECURITY.md) for the private vulnerability reporting path.

## License

`github-token-broker` is dual-licensed under the [Apache License 2.0](LICENSE-APACHE) or the [MIT License](LICENSE-MIT), at your option.

Unless you explicitly state otherwise, any contribution intentionally submitted for inclusion in this project shall be dual-licensed as above, without any additional terms or conditions.
