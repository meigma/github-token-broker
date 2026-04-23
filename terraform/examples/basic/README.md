# Basic example

Smallest invocation of the module. Assumes the three SSM parameters already exist at the default paths under `/github-token-broker/app/`:

- `/github-token-broker/app/client-id`
- `/github-token-broker/app/installation-id`
- `/github-token-broker/app/private-key-pem` (SecureString)

## Usage

```sh
cp terraform.tfvars.example terraform.tfvars
# edit terraform.tfvars with your values
tofu init  # or: terraform init
tofu apply
```

The `gh` CLI must be installed and authenticated on the machine running `apply`; the module uses it to download the release asset.

After apply, invoke the function:

```sh
aws lambda invoke --function-name github-token-broker --payload '{}' --cli-binary-format raw-in-base64-out /tmp/out.json
cat /tmp/out.json
```

A healthy response is a JSON object with `token`, `expires_at`, `repositories`, and `permissions`.
