# Bootstrap SSM parameters alongside the broker

First-time-setup example that can (optionally) create the three SSM parameters in the same apply as the Lambda. Split out into its own example so production users don't accidentally manage sensitive values through Terraform state.

## Why this is separate

The GitHub App private key is a secret. When `create_ssm_parameters = true`, the PEM value is passed through `aws_ssm_parameter` and ends up **in plaintext inside Terraform state**. That is acceptable for a first-time bootstrap in a non-production account, but it is **not** where production secrets should live. In production:

- Create the parameters out-of-band (AWS Console, `aws ssm put-parameter`, SOPS, a secret-manager pipeline, etc.).
- Leave `create_ssm_parameters = false` (the default).
- Use `examples/basic/` to provision the Lambda against the pre-existing parameters.

## Usage (first-time bootstrap)

```sh
cp terraform.tfvars.example terraform.tfvars
# edit terraform.tfvars: set create_ssm_parameters = true and populate
# github_app_* values, ideally from an environment-injected source.
tofu init
tofu apply
```

After the first apply, remove the sensitive values from `terraform.tfvars`, set `create_ssm_parameters = false`, and import or re-home the parameters to an out-of-state workflow.
