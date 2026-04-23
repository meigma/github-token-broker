terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0, < 7.0"
    }
  }
}

provider "aws" {
  region = var.region
}

locals {
  ssm_parameter_paths = {
    client_id       = "/github-token-broker/app/client-id"
    installation_id = "/github-token-broker/app/installation-id"
    private_key     = "/github-token-broker/app/private-key-pem"
  }
}

resource "aws_ssm_parameter" "client_id" {
  count = var.create_ssm_parameters ? 1 : 0

  name  = local.ssm_parameter_paths.client_id
  type  = "String"
  value = var.github_app_client_id
}

resource "aws_ssm_parameter" "installation_id" {
  count = var.create_ssm_parameters ? 1 : 0

  name  = local.ssm_parameter_paths.installation_id
  type  = "String"
  value = var.github_app_installation_id
}

resource "aws_ssm_parameter" "private_key" {
  count = var.create_ssm_parameters ? 1 : 0

  name   = local.ssm_parameter_paths.private_key
  type   = "SecureString"
  value  = var.github_app_private_key_pem
  key_id = var.kms_key_id
}

module "broker" {
  source = "../.."

  function_name    = "github-token-broker"
  repository_owner = var.repository_owner
  repository_name  = var.repository_name

  lambda_artifact = {
    release_version = var.release_version
  }

  ssm_parameter_paths = local.ssm_parameter_paths
  kms_key_arn         = var.kms_key_arn

  depends_on = [
    aws_ssm_parameter.client_id,
    aws_ssm_parameter.installation_id,
    aws_ssm_parameter.private_key,
  ]
}
