variable "region" {
  description = "AWS region."
  type        = string
}

variable "repository_owner" {
  description = "GitHub owner the broker issues tokens for."
  type        = string
}

variable "repository_name" {
  description = "GitHub repository the broker issues tokens for."
  type        = string
}

variable "release_version" {
  description = "Upstream release tag to deploy (e.g. v1.0.0)."
  type        = string
}

variable "invoker_principal_arn" {
  description = "IAM principal ARN allowed to invoke the Function URL."
  type        = string
}
