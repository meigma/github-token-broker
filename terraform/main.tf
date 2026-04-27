resource "null_resource" "fetch_release" {
  count = local.use_release_asset ? 1 : 0

  triggers = {
    release_version    = var.lambda_artifact.release_version
    release_repository = var.release_repository
    cache_dir          = local.release_cache_dir
  }

  provisioner "local-exec" {
    interpreter = ["/bin/bash", "-c"]
    environment = {
      RELEASE_VERSION    = coalesce(var.lambda_artifact.release_version, "")
      RELEASE_REPOSITORY = var.release_repository
      RELEASE_CACHE_DIR  = coalesce(local.release_cache_dir, "")
    }
    command = <<-EOT
      set -euo pipefail

      mkdir -p "$RELEASE_CACHE_DIR"

      if ! command -v gh >/dev/null 2>&1; then
        echo "error: gh CLI is required to fetch the release asset. Install it or pre-download and pass lambda_zip_path." >&2
        exit 1
      fi

      if ! command -v sha256sum >/dev/null 2>&1; then
        echo "error: sha256sum is required to verify the release asset." >&2
        exit 1
      fi

      gh release download "$RELEASE_VERSION" \
        --repo "$RELEASE_REPOSITORY" \
        --pattern github-token-broker.zip \
        --pattern checksums.txt \
        --dir "$RELEASE_CACHE_DIR" \
        --clobber

      (cd "$RELEASE_CACHE_DIR" && sha256sum --check --status checksums.txt)
    EOT
  }
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = local.log_group_name
  retention_in_days = var.log_retention_days == 0 ? null : var.log_retention_days
  tags              = local.module_tags
}

resource "aws_lambda_function" "broker" {
  function_name = var.function_name
  role          = aws_iam_role.lambda.arn
  runtime       = "provided.al2023"
  handler       = "bootstrap"
  architectures = [var.architecture]
  memory_size   = var.memory_size
  timeout       = var.timeout

  filename         = local.lambda_filename
  source_code_hash = local.lambda_source_code_hash

  s3_bucket = local.use_s3_source ? var.lambda_artifact.lambda_source_s3.bucket : null
  s3_key    = local.use_s3_source ? var.lambda_artifact.lambda_source_s3.key : null

  environment {
    variables = local.environment
  }

  tags = local.module_tags

  depends_on = [
    null_resource.fetch_release,
    aws_cloudwatch_log_group.lambda,
    aws_iam_role_policy.lambda,
  ]
}
