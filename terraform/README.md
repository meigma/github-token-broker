# github-token-broker — Terraform module

Deploys [`github-token-broker`](https://github.com/meigma/github-token-broker) as an AWS Lambda, sourced from a published GitHub Release asset.

The module is opinionated in what matters for supply-chain integrity (SHA256 verification on download, least-privilege IAM, `AWS_IAM`-only Function URL) and configurable everywhere else (memory, timeout, tags, log retention, permissions set, SSM parameter paths, KMS).

## Usage

```hcl
module "broker" {
  source = "github.com/meigma/github-token-broker//terraform?ref=v1.0.0"

  function_name    = "github-token-broker"
  repository_owner = "example-org"
  repository_name  = "example-repo"

  lambda_artifact = {
    release_version = "v1.0.0"
  }
}
```

Three ways to source the Lambda zip via `lambda_artifact`:

| Field | When to use |
| --- | --- |
| `release_version = "v1.0.0"` | Normal case. The module runs `gh release download` on the `terraform apply` host, verifies the zip against `checksums.txt`, and caches the zip under `.terraform/github-token-broker/<function>/<version>/`. Requires `gh` and `sha256sum` on PATH. |
| `lambda_zip_path = "/path/to/github-token-broker.zip"` | Air-gapped or pre-downloaded workflows where `gh` is unavailable at apply time. Verify the zip out-of-band with `gh attestation verify` before using this path. |
| `lambda_source_s3 = { bucket = "...", key = "..." }` | When the zip is already staged to S3 (e.g. by CI). The module references S3 directly; no local download. |

Exactly one of the three must be set; a validation rule enforces this.

See [`examples/basic`](./examples/basic) for the smallest viable config, [`examples/function-url`](./examples/function-url) for a Function URL deployment, and [`examples/with-ssm-bootstrap`](./examples/with-ssm-bootstrap) for first-time SSM parameter creation.

## Verification

Inline SHA256 verification happens on every apply that downloads the release asset. `checksums.txt` is signed as part of the SLSA build provenance attestation, so the trust anchor is the attestation, not the checksum file on its own.

Run `gh attestation verify` before deploying a new version pin to confirm the asset's provenance:

```sh
TAG=v1.0.0
gh release download "$TAG" -R meigma/github-token-broker -p github-token-broker.zip
gh attestation verify github-token-broker.zip \
  --repo meigma/github-token-broker \
  --signer-workflow meigma/github-token-broker/.github/workflows/reusable-release.yml \
  --source-ref "refs/tags/$TAG" \
  --deny-self-hosted-runners
```

The module does not invoke `gh attestation verify` itself — Terraform has no ergonomic way to run it inline. Treat it as a pre-deployment check in your CI or your human review loop.

## Sandbox validation

End-to-end validation requires an AWS account and a GitHub App with an installation covering `repository_owner/repository_name`. A minimal procedure:

1. Create a GitHub App with `contents: read` (or whatever `permissions` you pass), install it on the target repo, and record the client ID, installation ID, and private key PEM.
2. Put the three values in SSM:
   ```sh
   aws ssm put-parameter --name /github-token-broker/app/client-id --type String --value "Iv23li..."
   aws ssm put-parameter --name /github-token-broker/app/installation-id --type String --value "12345678"
   aws ssm put-parameter --name /github-token-broker/app/private-key-pem --type SecureString --value "$(cat key.pem)"
   ```
3. Apply `examples/basic`:
   ```sh
   cd examples/basic
   cp terraform.tfvars.example terraform.tfvars
   # edit values
   terraform init && terraform apply
   ```
4. Invoke the Lambda:
   ```sh
   aws lambda invoke --function-name github-token-broker --payload '{}' \
     --cli-binary-format raw-in-base64-out /tmp/out.json
   jq . /tmp/out.json
   ```
5. Confirm the response contains `token`, `expires_at`, `repositories`, and `permissions`. Use the token against the GitHub API to confirm it works:
   ```sh
   TOKEN=$(jq -r .token /tmp/out.json)
   gh api "repos/<owner>/<repo>" -H "Authorization: token $TOKEN" | jq .name
   ```

## Security notes

- IAM policy grants only `ssm:GetParameters` on the three configured paths, `kms:Decrypt` on the explicit CMK ARN when provided, and `logs:CreateLogStream`/`logs:PutLogEvents` on the module-managed log group. No wildcards on sensitive actions.
- Function URLs are always `AWS_IAM`-authenticated. The module will not create a `NONE`-auth URL.
- The Lambda rejects non-empty invocation payloads, so the deployed `permissions` set is the upper bound — callers cannot request more.
- `AWS_REGION` is provided by the Lambda runtime automatically; the module does not set it explicitly.
- `CHANGELOG.md` and the release page are the source of truth for what's in a given `release_version`. The module performs SHA256 verification against `checksums.txt`, not signature verification — use `gh attestation verify` for supply-chain assurance.

## Requirements on the apply host (default mode)

- `gh` (≥ 2.40) — authenticated against the target release repository.
- `sha256sum` — present on most Linux distros and macOS with coreutils. The `null_resource` aborts early if either binary is missing.

Switch to `lambda_zip_path` or `lambda_source_s3` if the apply host cannot satisfy these.

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.6 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 5.0, < 7.0 |
| <a name="requirement_null"></a> [null](#requirement\_null) | >= 3.2 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | >= 5.0, < 7.0 |
| <a name="provider_null"></a> [null](#provider\_null) | >= 3.2 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_cloudwatch_log_group.lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_log_group) | resource |
| [aws_iam_role.lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy.lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy) | resource |
| [aws_lambda_function.broker](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lambda_function) | resource |
| [aws_lambda_function_url.broker](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lambda_function_url) | resource |
| [null_resource.fetch_release](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource) | resource |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_iam_policy_document.assume_role](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_iam_policy_document.lambda](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_partition.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/partition) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_architecture"></a> [architecture](#input\_architecture) | Lambda architecture. arm64 matches the published release zip. | `string` | `"arm64"` | no |
| <a name="input_enable_function_url"></a> [enable\_function\_url](#input\_enable\_function\_url) | Create a Lambda Function URL with AWS\_IAM auth. Never creates a NONE-auth URL. | `bool` | `false` | no |
| <a name="input_function_name"></a> [function\_name](#input\_function\_name) | Name of the Lambda function. | `string` | n/a | yes |
| <a name="input_github_api_base_url"></a> [github\_api\_base\_url](#input\_github\_api\_base\_url) | GitHub API base URL. Override for GitHub Enterprise Server. | `string` | `"https://api.github.com"` | no |
| <a name="input_kms_key_arn"></a> [kms\_key\_arn](#input\_kms\_key\_arn) | KMS key ARN used by SSM to encrypt the private key parameter. Set only when the customer uses a CMK instead of the AWS-managed key. Null disables kms:Decrypt in the role policy. | `string` | `null` | no |
| <a name="input_lambda_artifact"></a> [lambda\_artifact](#input\_lambda\_artifact) | Source of the Lambda zip. Exactly one of the three fields must be set:<br/><br/>- `release_version`: a tag published on `release_repository` (e.g. "v1.0.0").<br/>  The module downloads `github-token-broker.zip` and `checksums.txt` via the<br/>  `gh` CLI on the machine running `terraform apply`, verifies the zip's<br/>  SHA256 against `checksums.txt`, and points the Lambda at the cached copy.<br/>- `lambda_zip_path`: absolute path to a pre-downloaded zip. Used for<br/>  air-gapped workflows where `gh` is unavailable at apply time.<br/>- `lambda_source_s3`: S3 bucket/key holding the zip. Used when the zip is<br/>  staged to S3 out-of-band (e.g. by CI).<br/><br/>Inline SHA256 verification is defense-in-depth against a corrupted<br/>download. It is NOT a replacement for `gh attestation verify`, which is<br/>the canonical supply-chain check. See `terraform/README.md` for guidance. | <pre>object({<br/>    release_version = optional(string)<br/>    lambda_zip_path = optional(string)<br/>    lambda_source_s3 = optional(object({<br/>      bucket = string<br/>      key    = string<br/>    }))<br/>  })</pre> | n/a | yes |
| <a name="input_log_level"></a> [log\_level](#input\_log\_level) | Slog level. One of debug, info, warn, error. | `string` | `"info"` | no |
| <a name="input_log_retention_days"></a> [log\_retention\_days](#input\_log\_retention\_days) | CloudWatch Logs retention in days. | `number` | `30` | no |
| <a name="input_memory_size"></a> [memory\_size](#input\_memory\_size) | Lambda memory in MB. | `number` | `128` | no |
| <a name="input_permissions"></a> [permissions](#input\_permissions) | Repository permissions requested on each minted token. Serialized to GITHUB\_TOKEN\_BROKER\_PERMISSIONS as JSON. | `map(string)` | <pre>{<br/>  "contents": "read"<br/>}</pre> | no |
| <a name="input_release_repository"></a> [release\_repository](#input\_release\_repository) | GitHub repository to pull the release asset from when lambda\_artifact.release\_version is set. Defaults to the upstream repo. | `string` | `"meigma/github-token-broker"` | no |
| <a name="input_repository_name"></a> [repository\_name](#input\_repository\_name) | GitHub repository the broker issues tokens for. | `string` | n/a | yes |
| <a name="input_repository_owner"></a> [repository\_owner](#input\_repository\_owner) | GitHub owner of the repository the broker issues tokens for. | `string` | n/a | yes |
| <a name="input_ssm_parameter_paths"></a> [ssm\_parameter\_paths](#input\_ssm\_parameter\_paths) | SSM parameter paths holding the GitHub App credentials. All paths must be absolute. | <pre>object({<br/>    client_id       = string<br/>    installation_id = string<br/>    private_key     = string<br/>  })</pre> | <pre>{<br/>  "client_id": "/github-token-broker/app/client-id",<br/>  "installation_id": "/github-token-broker/app/installation-id",<br/>  "private_key": "/github-token-broker/app/private-key-pem"<br/>}</pre> | no |
| <a name="input_tags"></a> [tags](#input\_tags) | Tags applied to all resources created by this module. | `map(string)` | `{}` | no |
| <a name="input_timeout"></a> [timeout](#input\_timeout) | Lambda execution timeout in seconds. | `number` | `10` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_deployed_version"></a> [deployed\_version](#output\_deployed\_version) | Release version actually deployed, or null when the module was pointed at a local zip or S3 source. |
| <a name="output_function_arn"></a> [function\_arn](#output\_function\_arn) | ARN of the Lambda function. |
| <a name="output_function_invoke_arn"></a> [function\_invoke\_arn](#output\_function\_invoke\_arn) | Invoke ARN, suitable for API Gateway or EventBridge integrations. |
| <a name="output_function_name"></a> [function\_name](#output\_function\_name) | Name of the Lambda function. |
| <a name="output_function_url"></a> [function\_url](#output\_function\_url) | Function URL when enable\_function\_url is true; null otherwise. |
| <a name="output_log_group_name"></a> [log\_group\_name](#output\_log\_group\_name) | Name of the CloudWatch Log Group backing Lambda logs. |
| <a name="output_role_arn"></a> [role\_arn](#output\_role\_arn) | ARN of the Lambda execution role. |
| <a name="output_role_name"></a> [role\_name](#output\_role\_name) | Name of the Lambda execution role. |
<!-- END_TF_DOCS -->
