data "aws_region" "current" {}

data "aws_caller_identity" "current" {}

data "aws_partition" "current" {}

locals {
  account_id = data.aws_caller_identity.current.account_id
  partition  = data.aws_partition.current.partition
  region     = data.aws_region.current.name

  ssm_parameter_arns = [
    for path in [
      var.ssm_parameter_paths.client_id,
      var.ssm_parameter_paths.installation_id,
      var.ssm_parameter_paths.private_key,
    ] :
    "arn:${local.partition}:ssm:${local.region}:${local.account_id}:parameter${path}"
  ]

  log_group_name = "/aws/lambda/${var.function_name}"
  log_group_arn  = "arn:${local.partition}:logs:${local.region}:${local.account_id}:log-group:${local.log_group_name}"

  use_release_asset = try(var.lambda_artifact.release_version, null) != null
  use_local_zip     = try(var.lambda_artifact.lambda_zip_path, null) != null
  use_s3_source     = try(var.lambda_artifact.lambda_source_s3, null) != null

  release_cache_dir = local.use_release_asset ? (
    "${path.root}/.terraform/github-token-broker/${var.function_name}/${var.lambda_artifact.release_version}"
  ) : null

  release_zip_path = local.use_release_asset ? "${local.release_cache_dir}/github-token-broker.zip" : null

  lambda_filename = coalesce(
    local.use_release_asset ? local.release_zip_path : null,
    local.use_local_zip ? var.lambda_artifact.lambda_zip_path : null,
  )

  lambda_source_code_hash = local.use_release_asset ? sha256(var.lambda_artifact.release_version) : null

  environment = merge(
    {
      GITHUB_TOKEN_BROKER_REPOSITORY_OWNER      = var.repository_owner
      GITHUB_TOKEN_BROKER_REPOSITORY_NAME       = var.repository_name
      GITHUB_TOKEN_BROKER_CLIENT_ID_PARAM       = var.ssm_parameter_paths.client_id
      GITHUB_TOKEN_BROKER_INSTALLATION_ID_PARAM = var.ssm_parameter_paths.installation_id
      GITHUB_TOKEN_BROKER_PRIVATE_KEY_PARAM     = var.ssm_parameter_paths.private_key
      GITHUB_TOKEN_BROKER_GITHUB_API_BASE_URL   = var.github_api_base_url
      GITHUB_TOKEN_BROKER_LOG_LEVEL             = var.log_level
      GITHUB_TOKEN_BROKER_PERMISSIONS           = jsonencode(var.permissions)
    },
  )

  module_tags = merge(
    {
      "github-token-broker:managed-by" = "terraform"
      "github-token-broker:module"     = "meigma/github-token-broker"
    },
    var.tags,
  )
}
