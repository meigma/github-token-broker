#!/usr/bin/env bash
# Confirm that release-please created a draft release matching the pushed tag.
# Fails early if the tag does not resolve to a draft release, so the tag
# workflow bails before doing any build work.

set -euo pipefail

: "${GITHUB_REF_NAME:?GITHUB_REF_NAME must be set by the GitHub Actions runtime}"
: "${GH_TOKEN:?GH_TOKEN must be set (pass in secrets.GITHUB_TOKEN)}"

tag="${GITHUB_REF_NAME}"

attempt_max=6
attempt_sleep=10

for attempt in $(seq 1 "${attempt_max}"); do
  if release_json="$(gh release view "${tag}" --json tagName,isDraft,url 2>/dev/null)"; then
    tag_name="$(jq -r '.tagName' <<<"${release_json}")"
    is_draft="$(jq -r '.isDraft' <<<"${release_json}")"
    release_url="$(jq -r '.url' <<<"${release_json}")"

    printf 'draft release: %s\n' "${release_url}"

    if [[ "${tag_name}" != "${tag}" ]]; then
      printf 'release tag mismatch: expected %s, got %s\n' "${tag}" "${tag_name}" >&2
      exit 1
    fi

    if [[ "${is_draft}" != "true" ]]; then
      printf 'release %s is not a draft release\n' "${tag}" >&2
      exit 1
    fi

    exit 0
  fi

  printf 'release %s not visible yet (attempt %d/%d); sleeping %ds\n' \
    "${tag}" "${attempt}" "${attempt_max}" "${attempt_sleep}"
  sleep "${attempt_sleep}"
done

printf 'release %s did not appear within %d attempts\n' "${tag}" "${attempt_max}" >&2
exit 1
