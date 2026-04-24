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

## Quick Start

Deploy the Lambda by pinning the first-party Terraform module from git:

```hcl
module "broker" {
  source = "github.com/meigma/github-token-broker//terraform?ref=v1.1.0"

  function_name    = "github-token-broker"
  repository_owner = "your-org"
  repository_name  = "your-repo"

  lambda_artifact = {
    release_version = "v1.1.0"
  }
}
```

Apply, then invoke with `aws lambda invoke --payload 'null'`. Walk through the full setup — GitHub App, SSM parameters, invocation — in the [Deploy your first broker tutorial](docs/docs/tutorials/deploy-your-first-broker.md).

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

## Verification

Releases ship the Lambda zip alongside a `checksums.txt` (SHA256). Build provenance and an SBOM are persisted to GitHub's Attestations API; verify them with [`gh attestation verify`](https://cli.github.com/manual/gh_attestation_verify) rather than downloading signature or SBOM files from the release page.

```sh
TAG=v1.1.0
gh release download "$TAG" -R meigma/github-token-broker \
  -p 'github-token-broker.zip' -p 'checksums.txt'

sha256sum --check checksums.txt

gh release verify "$TAG" -R meigma/github-token-broker
gh release verify-asset "$TAG" ./github-token-broker.zip -R meigma/github-token-broker

gh attestation verify ./github-token-broker.zip \
  --repo meigma/github-token-broker \
  --signer-workflow meigma/github-token-broker/.github/workflows/reusable-release.yml \
  --source-ref "refs/tags/$TAG" \
  --deny-self-hosted-runners
```

The `sha256sum` check is defense-in-depth against a corrupted download; `gh attestation verify` is the canonical supply-chain check. `checksums.txt` itself is bound to the provenance attestation, so anchor trust in the attestation rather than the file alone.

The attestation call above validates the SLSA build provenance by default. To validate the SBOM attestation specifically, add a predicate filter:

```sh
gh attestation verify ./github-token-broker.zip \
  --repo meigma/github-token-broker \
  --predicate-type https://spdx.dev/Document
```

For air-gapped or offline verification, download the attestation bundle first and pass it explicitly:

```sh
gh attestation download ./github-token-broker.zip -R meigma/github-token-broker
gh attestation verify ./github-token-broker.zip \
  --bundle github-token-broker.zip.bundle.jsonl \
  --signer-workflow meigma/github-token-broker/.github/workflows/reusable-release.yml \
  --source-ref "refs/tags/$TAG" \
  --deny-self-hosted-runners
```

See [docs/explanation/release-architecture.md](docs/docs/explanation/release-architecture.md) for the full pipeline design and the rationale behind the attestation-only verification channel.

## Documentation

Full documentation is published at <https://github-token-broker.meigma.dev>. The source lives under [`docs/`](docs/) and is organized by [Diátaxis](https://diataxis.fr/) quadrant:

- [Tutorial: deploy your first broker](https://github-token-broker.meigma.dev/tutorials/deploy-your-first-broker)
- [How-to guides](https://github-token-broker.meigma.dev/how-to/rotate-github-app-private-key) — rotate the private key, change target repo, use with GitHub Enterprise Server.
- [Reference](https://github-token-broker.meigma.dev/reference/environment-variables) — env vars, response schema, IAM policy, SSM parameters, error messages.
- [Explanation](https://github-token-broker.meigma.dev/explanation/architecture) — architecture diagrams, security model, design rationale.

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
