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

variable "create_ssm_parameters" {
  description = <<-EOT
    When true, this example creates the three SSM parameters holding the
    GitHub App credentials. Leave false in production: the sensitive
    private key would be stored in Terraform state in plaintext. Manage
    the parameters out-of-band (e.g. via SOPS or a secret-manager
    pipeline) and keep this flag off.
  EOT
  type        = bool
  default     = false
}

variable "github_app_client_id" {
  description = "GitHub App client ID. Only used when create_ssm_parameters is true."
  type        = string
  default     = null
}

variable "github_app_installation_id" {
  description = "GitHub App installation ID. Only used when create_ssm_parameters is true."
  type        = string
  default     = null
}

variable "github_app_private_key_pem" {
  description = <<-EOT
    GitHub App private key PEM. Only used when create_ssm_parameters is
    true. Treat this value as a secret — it lands in Terraform state in
    plaintext if provided here.
  EOT
  type        = string
  default     = null
  sensitive   = true
}

variable "kms_key_id" {
  description = "KMS key ID/alias used to encrypt the private-key SSM SecureString."
  type        = string
  default     = null
}

variable "kms_key_arn" {
  description = "KMS key ARN passed to the broker for kms:Decrypt permissions."
  type        = string
  default     = null
}
