# Function URL example

Provisions the broker with a Lambda Function URL protected by `AWS_IAM` authorization and an explicit `aws_lambda_permission` scoping which principal can call it. The module never creates `NONE`-auth URLs.

## Usage

```sh
cp terraform.tfvars.example terraform.tfvars
# edit terraform.tfvars — invoker_principal_arn must be the IAM role/user that will call the URL
tofu init
tofu apply
```

Callers must sign requests with SigV4; a typical caller uses the AWS SDK's Lambda URL signer or `awscurl`.

```sh
awscurl --service lambda --region "$REGION" "$(tofu output -raw function_url)"
```

If you see `Forbidden`, confirm your caller identity matches `invoker_principal_arn` and that `aws_lambda_permission` propagated (a short eventual-consistency window is normal after first apply).
