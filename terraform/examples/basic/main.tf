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

module "broker" {
  source = "../.."

  function_name    = "github-token-broker"
  repository_owner = var.repository_owner
  repository_name  = var.repository_name

  lambda_artifact = {
    release_version = var.release_version
  }
}
