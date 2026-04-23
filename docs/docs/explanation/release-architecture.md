---
title: Release Architecture
description: How github-token-broker versions, builds, signs, and publishes releases.
---

# Release Architecture

`github-token-broker` splits release work across four workflow pieces so that
versioning, publication, and attestation each have clear ownership and a
clear failure boundary. This is the pattern the [publishing
skill](https://github.com/anthropics/claude-code) baseline recommends, and it
is what downstream consumers rely on when they run `gh attestation verify`.

## The four pieces

1. **Versioning** — [`.github/workflows/release-please.yml`](https://github.com/meigma/github-token-broker/blob/master/.github/workflows/release-please.yml).
   On every push to `master`, release-please reads the Conventional Commit
   history and updates (or creates) an open release PR. Merging that PR
   creates the `vX.Y.Z` tag and a **draft** GitHub Release.
2. **Tag entrypoint** — [`.github/workflows/release.yml`](https://github.com/meigma/github-token-broker/blob/master/.github/workflows/release.yml).
   Triggered by the tag push. Three jobs in order:
   - `verify-draft-release` — confirms the draft release exists for the tag
     and bails early on mismatch.
   - `release` — calls the reusable publish workflow.
   - `publish-draft-release` — flips the release from draft to published
     only after the reusable workflow succeeds.
3. **Reusable publish** — [`.github/workflows/reusable-release.yml`](https://github.com/meigma/github-token-broker/blob/master/.github/workflows/reusable-release.yml).
   The trusted builder. Builds the Lambda zip, generates a checksum file
   and an SBOM, creates GitHub Artifact Attestations for both, and uploads
   the zip to the draft release. Its identity is what
   `gh attestation verify --signer-workflow` validates.
4. **Continuous security** — CodeQL, dependency review, and OpenSSF
   Scorecard workflows. They keep the release path honest between tagged
   releases.

## What ships on a release

Only the Lambda zip. That's it.

Provenance and the SBOM are pushed to [GitHub's Attestations
API](https://docs.github.com/en/actions/security-guides/using-artifact-attestations-to-establish-provenance-for-builds)
and fetched on demand by `gh attestation verify`. They are **not** attached
as release assets.

This design follows the publishing skill's rule: attestation metadata is
pipeline handoff data, not release payload. Attaching provenance bundles or
SBOM files to the release would duplicate what the Attestations API already
owns — and would invite consumers to verify the wrong thing (a local file
they downloaded) instead of the right thing (an API-resident attestation
bound to a specific digest and workflow identity).

## Why no Cosign

GitHub Artifact Attestations is a keyless Sigstore flow under the hood.
Adding a separate `cosign sign-blob` step produces a second signature over
the same content with a second identity, and yields a second bundle for
consumers to verify. That is pure duplication and a second key-management
surface.

If a downstream consumer ever needs bare Sigstore bundles, GitHub's
attestation bundle is available via `gh attestation download`. See the
[Verification section of the README](https://github.com/meigma/github-token-broker#verification)
for the offline flow.

## Why a separate reusable workflow

`gh attestation verify --signer-workflow` pins attestations to a specific
workflow file's identity. Splitting the reusable publisher into its own
file gives the signer a stable, minimal identity that is distinct from the
tag-entrypoint workflow that called it. This is the SLSA Level 3 trusted
builder property — if `release.yml` is ever compromised or rewritten, the
reusable file's identity (and therefore the attestations consumers have
already verified) does not change.

## How this differs from a typical GoReleaser flow

Many Go projects drive releases with GoReleaser + Cosign + per-archive
SBOMs attached to the release. That pattern is a reasonable default for
multi-platform binary distributions where consumers expect to download
artifacts from the release page and verify signatures side by side.

For this repo the Lambda zip is the only artifact and GitHub's Attestations
API is the verification channel, so we skip GoReleaser, Cosign, and
on-release SBOMs. The publishing skill's rule — "do not make checksums,
digests, SBOMs, and attestation metadata release assets unless the project
explicitly supports offline verification" — is what the current pipeline
follows.

## Execution order for a real release

1. Merge feature PRs with Conventional Commit titles. Release-please
   updates the open release PR on each push to `master`.
2. Merge the release PR. Release-please creates the tag and the draft
   release.
3. `release.yml` fires on the tag push. If any step fails (`verify`,
   `release`, or `publish-draft-release`), the release stays in draft and
   is effectively invisible to consumers.
4. Fix forward. Tags are immutable, so a broken tagged release is resolved
   by the next Conventional Commit rolling a new version.

## Further reading

- [Using artifact attestations to establish provenance for builds](https://docs.github.com/en/actions/security-guides/using-artifact-attestations-to-establish-provenance-for-builds)
- [`gh attestation verify`](https://cli.github.com/manual/gh_attestation_verify)
- [SLSA — Build Levels](https://slsa.dev/spec/v1.0/levels)
- [release-please](https://github.com/googleapis/release-please)
