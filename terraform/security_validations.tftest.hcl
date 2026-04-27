mock_provider "aws" {
  mock_data "aws_caller_identity" {
    defaults = {
      account_id = "123456789012"
    }
  }

  mock_data "aws_partition" {
    defaults = {
      partition = "aws"
    }
  }

  mock_data "aws_region" {
    defaults = {
      region = "us-east-1"
    }
  }

  mock_data "aws_iam_policy_document" {
    defaults = {
      json = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":\"sts:AssumeRole\",\"Principal\":{\"Service\":\"lambda.amazonaws.com\"}}]}"
    }
  }
}

mock_provider "null" {}

variables {
  function_name    = "github-token-broker"
  repository_owner = "acme"
  repository_name  = "widgets"

  lambda_artifact = {
    release_version = "v1.1.0"
  }
}

run "reject_function_name_shell_metacharacters" {
  command = plan

  variables {
    function_name = "broker';touch/tmp/pwn"
  }

  expect_failures = [
    var.function_name,
  ]
}

run "reject_release_repository_shell_metacharacters" {
  command = plan

  variables {
    release_repository = "meigma/github-token-broker';touch/tmp/pwn"
  }

  expect_failures = [
    var.release_repository,
  ]
}

run "reject_repository_owner_path_escape" {
  command = plan

  variables {
    repository_owner = "acme/widgets"
  }

  expect_failures = [
    var.repository_owner,
  ]
}

run "reject_repository_name_percent_escape" {
  command = plan

  variables {
    repository_name = "widgets%2fadmin"
  }

  expect_failures = [
    var.repository_name,
  ]
}

run "reject_wildcard_ssm_paths" {
  command = plan

  variables {
    ssm_parameter_paths = {
      client_id       = "/github-token-broker/app/client-id"
      installation_id = "/github-token-broker/app/*"
      private_key     = "/github-token-broker/app/private-key-pem"
    }
  }

  expect_failures = [
    var.ssm_parameter_paths,
  ]
}

run "reject_wildcard_kms_key_arn" {
  command = plan

  variables {
    kms_key_arn = "arn:aws:kms:us-east-1:123456789012:key/*"
  }

  expect_failures = [
    var.kms_key_arn,
  ]
}
