---
title: Errors
sidebar_position: 5
description: Error messages the broker surfaces, their causes, and how to resolve them.
---

# Errors

All broker errors are returned as Lambda function errors (not as a `200` with an error body). The AWS CLI surfaces them as a `FunctionError` field on the invoke response.

| Error message | Cause | Resolution |
|---|---|---|
| `github-token-broker does not accept invocation input` | Payload was not empty or literal `null`. `{}` triggers this. | Invoke with `--payload 'null'` or omit the payload. See [why-empty-payloads](../explanation/why-empty-payloads). |
| `AWS_REGION is required` | The Lambda started without a region in the environment. | Should never happen under the Lambda runtime. If reproducing locally, set `AWS_REGION`. |
| `GITHUB_TOKEN_BROKER_REPOSITORY_OWNER is required` | Required config missing. | Set the Terraform module's `repository_owner` input. |
| `GITHUB_TOKEN_BROKER_REPOSITORY_NAME is required` | Required config missing. | Set the Terraform module's `repository_name` input. |
| `GITHUB_TOKEN_BROKER_CLIENT_ID_PARAM must be an absolute SSM parameter path` | SSM path does not start with `/`. Applies to all three parameter-path variables. | Use an absolute path in the module's `ssm_parameter_paths` input. |
| `GITHUB_TOKEN_BROKER_PERMISSIONS must be a JSON object of string-to-string entries` | `GITHUB_TOKEN_BROKER_PERMISSIONS` is not valid JSON or has non-string values. | Supply a JSON object like `{"contents":"read"}`. |
| `missing GitHub App SSM parameters: [...]` | One or more SSM parameters do not exist at the configured paths. | Create the parameters (see [SSM parameter shapes](./ssm-parameter-shapes)), or fix the paths. |
| SSM `AccessDeniedException` | The Lambda's role lacks `ssm:GetParameters` on the parameter ARNs, or `kms:Decrypt` on the CMK. | Verify the module's IAM policy matches the parameter paths and CMK. See [IAM permissions](./iam-permissions). |
| JWT signing failure / PEM parse error | The private-key parameter value is not a valid PEM. Common causes: line endings mangled during `put-parameter`, wrong parameter updated, truncated file. | Re-upload the PEM with `$(cat key.pem)` to preserve content. See [rotate-github-app-private-key](../how-to/rotate-github-app-private-key). |
| GitHub `401 Unauthorized` on `/app/installations/{id}/access_tokens` | The private key does not match the App's active keys, or the client ID does not match the App. | Verify the PEM corresponds to a currently-active private key on the App, and the client ID is the App's client ID (not App ID). |
| GitHub `404 Not Found` on `/app/installations/{id}/access_tokens` | The installation ID is wrong, or the App was uninstalled from the repository. | Verify the installation ID, and that the App is installed on the target repo. |
| Broker error mentioning the target repository | The configured repository is not covered by the installation's repository selection. | Update the App's installation to include the repo, or change the target (see [change-target-repository](../how-to/change-target-repository)). |

## Where errors surface

- **AWS CLI**: `FunctionError` on `aws lambda invoke`; the message is in the response body.
- **CloudWatch Logs**: every failure logs an `ERROR` line with the message. The token is never logged on success or failure.
- **Caller code**: if invoking via the AWS SDK, errors surface as `Lambda.ServiceException`-family exceptions with the message in the `FunctionError` field.

## See also

- [Response schema](./response-schema) — what a successful response looks like.
- [IAM permissions](./iam-permissions) — the policy shape that avoids `AccessDenied`.
- [Rotate the GitHub App private key](../how-to/rotate-github-app-private-key) — fix for PEM-related failures.
