---
title: IAM permissions
sidebar_position: 3
description: The IAM statements the Lambda's execution role needs.
---

# IAM permissions

The Terraform module provisions the Lambda's execution role with a least-privilege inline policy. This page documents what the policy grants.

## What the module grants

### Read the three SSM parameters

```json
{
  "Sid": "ReadGitHubAppParameters",
  "Effect": "Allow",
  "Action": "ssm:GetParameters",
  "Resource": [
    "arn:aws:ssm:<region>:<account>:parameter/github-token-broker/app/client-id",
    "arn:aws:ssm:<region>:<account>:parameter/github-token-broker/app/installation-id",
    "arn:aws:ssm:<region>:<account>:parameter/github-token-broker/app/private-key-pem"
  ]
}
```

The actual parameter paths come from the module's `ssm_parameter_paths` input and default to the paths shown above. Only `ssm:GetParameters` (plural) is granted — the broker fetches all three in one batched call.

### Decrypt the private key (conditional)

Present only when the module's `kms_key_arn` variable is set:

```json
{
  "Sid": "DecryptPrivateKeyParameter",
  "Effect": "Allow",
  "Action": "kms:Decrypt",
  "Resource": "<kms_key_arn>"
}
```

When the SecureString parameter uses the AWS-managed SSM key (`alias/aws/ssm`), this statement is omitted; the AWS-managed key grants decrypt via SSM's service principal automatically.

### Write CloudWatch logs

```json
{
  "Sid": "WriteLambdaLogs",
  "Effect": "Allow",
  "Action": ["logs:CreateLogStream", "logs:PutLogEvents"],
  "Resource": "arn:aws:logs:<region>:<account>:log-group:/aws/lambda/<function_name>:*"
}
```

Scoped to the module-managed log group. `logs:CreateLogGroup` is **not** granted — the log group is created explicitly by the module, not by the Lambda at startup.

### Assume role policy

```json
{
  "Effect": "Allow",
  "Action": "sts:AssumeRole",
  "Principal": {"Service": "lambda.amazonaws.com"}
}
```

## What the module does **not** grant

- `ssm:GetParameter` (singular) — the broker uses batched `GetParameters`.
- `ssm:PutParameter` or any other write — the broker never modifies parameters.
- Wildcard SSM access — only the three specific parameter ARNs.
- `kms:Decrypt` when `kms_key_arn` is null — the AWS-managed SSM key handles that path implicitly.
- `logs:CreateLogGroup` — the log group is provisioned by Terraform.

## Caller-side IAM

Callers of the Lambda need `lambda:InvokeFunction` on the function ARN. That permission is out of scope for the broker module — attach it to the caller's role or user.

## See also

- [SSM parameter shapes](./ssm-parameter-shapes) — what the parameters actually store.
- [Security model](../explanation/security-model) — why the policy is shaped this way.
- [`terraform/iam.tf`](https://github.com/meigma/github-token-broker/blob/master/terraform/iam.tf) — canonical source for the policy.
