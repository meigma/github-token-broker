#!/usr/bin/env bash
# Flip the existing draft release to published. Runs only after all build
# and attestation steps have succeeded, so a broken release stays invisible
# until it is complete.

set -euo pipefail

: "${GITHUB_REF_NAME:?GITHUB_REF_NAME must be set by the GitHub Actions runtime}"
: "${GH_TOKEN:?GH_TOKEN must be set (pass in secrets.GITHUB_TOKEN)}"

gh release edit "${GITHUB_REF_NAME}" --draft=false
