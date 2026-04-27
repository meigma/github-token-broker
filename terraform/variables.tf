variable "function_name" {
  description = "Name of the Lambda function."
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9_-]{1,64}$", var.function_name))
    error_message = "function_name must be 1-64 characters and contain only letters, numbers, hyphens, and underscores."
  }
}

variable "repository_owner" {
  description = "GitHub owner of the repository the broker issues tokens for."
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9_.-]+$", var.repository_owner))
    error_message = "repository_owner must contain only letters, numbers, periods, underscores, and hyphens."
  }
}

variable "repository_name" {
  description = "GitHub repository the broker issues tokens for."
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9_.-]+$", var.repository_name))
    error_message = "repository_name must contain only letters, numbers, periods, underscores, and hyphens."
  }
}

variable "lambda_artifact" {
  description = <<-EOT
    Source of the Lambda zip. Exactly one of the three fields must be set:

    - `release_version`: a tag published on `release_repository` (e.g. "v1.0.0").
      The module downloads `github-token-broker.zip` and `checksums.txt` via the
      `gh` CLI on the machine running `terraform apply`, verifies the zip's
      SHA256 against `checksums.txt`, and points the Lambda at the cached copy.
    - `lambda_zip_path`: absolute path to a pre-downloaded zip. Used for
      air-gapped workflows where `gh` is unavailable at apply time.
    - `lambda_source_s3`: S3 bucket/key holding the zip. Used when the zip is
      staged to S3 out-of-band (e.g. by CI).

    Inline SHA256 verification is defense-in-depth against a corrupted
    download. It is NOT a replacement for `gh attestation verify`, which is
    the canonical supply-chain check. See `terraform/README.md` for guidance.
  EOT

  type = object({
    release_version = optional(string)
    lambda_zip_path = optional(string)
    lambda_source_s3 = optional(object({
      bucket = string
      key    = string
    }))
  })

  validation {
    condition = length(compact([
      try(var.lambda_artifact.release_version, null),
      try(var.lambda_artifact.lambda_zip_path, null),
      try(var.lambda_artifact.lambda_source_s3 == null ? null : "s3", null),
    ])) == 1
    error_message = "lambda_artifact must set exactly one of release_version, lambda_zip_path, or lambda_source_s3."
  }

  validation {
    condition = (
      try(var.lambda_artifact.release_version, null) == null ||
      can(regex("^v?[0-9]+\\.[0-9]+\\.[0-9]+(-[A-Za-z0-9.-]+)?$", var.lambda_artifact.release_version))
    )
    error_message = "lambda_artifact.release_version must be a semver tag such as \"v1.0.0\" or \"1.2.3-rc1\"."
  }

  validation {
    condition = (
      try(var.lambda_artifact.lambda_zip_path, null) == null ||
      length(trimspace(var.lambda_artifact.lambda_zip_path)) > 0
    )
    error_message = "lambda_artifact.lambda_zip_path must be non-empty when set."
  }
}

variable "release_repository" {
  description = "Literal OWNER/REPO GitHub repository to pull the release asset from when lambda_artifact.release_version is set. Defaults to the upstream repo."
  type        = string
  default     = "meigma/github-token-broker"

  validation {
    condition     = can(regex("^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$", var.release_repository))
    error_message = "release_repository must be a literal OWNER/REPO value containing only letters, numbers, periods, underscores, and hyphens."
  }
}

variable "permissions" {
  description = "Repository permissions requested on each minted token. Serialized to GITHUB_TOKEN_BROKER_PERMISSIONS as JSON."
  type        = map(string)
  default     = { contents = "read" }

  validation {
    condition     = length(var.permissions) > 0
    error_message = "permissions must request at least one permission."
  }

  validation {
    condition = alltrue([
      for k, v in var.permissions : length(trimspace(k)) > 0 && length(trimspace(v)) > 0
    ])
    error_message = "permissions entries must have non-empty keys and values."
  }
}

variable "ssm_parameter_paths" {
  description = "SSM parameter paths holding the GitHub App credentials. All paths must be absolute literal SSM paths."
  type = object({
    client_id       = string
    installation_id = string
    private_key     = string
  })
  default = {
    client_id       = "/github-token-broker/app/client-id"
    installation_id = "/github-token-broker/app/installation-id"
    private_key     = "/github-token-broker/app/private-key-pem"
  }

  validation {
    condition = alltrue([
      can(regex("^/[A-Za-z0-9_.\\-/]+$", var.ssm_parameter_paths.client_id)),
      can(regex("^/[A-Za-z0-9_.\\-/]+$", var.ssm_parameter_paths.installation_id)),
      can(regex("^/[A-Za-z0-9_.\\-/]+$", var.ssm_parameter_paths.private_key)),
    ])
    error_message = "ssm_parameter_paths entries must be absolute literal SSM paths containing only letters, numbers, periods, underscores, hyphens, and slashes."
  }
}

variable "github_api_base_url" {
  description = "GitHub API base URL. Override for GitHub Enterprise Server. The Lambda requires https except for loopback http URLs used in local tests."
  type        = string
  default     = "https://api.github.com"
}

variable "log_level" {
  description = "Slog level. One of debug, info, warn, error."
  type        = string
  default     = "info"

  validation {
    condition     = contains(["debug", "info", "warn", "error"], var.log_level)
    error_message = "log_level must be one of debug, info, warn, error."
  }
}

variable "architecture" {
  description = "Lambda architecture. arm64 matches the published release zip."
  type        = string
  default     = "arm64"

  validation {
    condition     = contains(["arm64", "x86_64"], var.architecture)
    error_message = "architecture must be arm64 or x86_64."
  }
}

variable "memory_size" {
  description = "Lambda memory in MB."
  type        = number
  default     = 128

  validation {
    condition     = var.memory_size >= 128 && var.memory_size <= 10240
    error_message = "memory_size must be between 128 and 10240 MB."
  }
}

variable "timeout" {
  description = "Lambda execution timeout in seconds."
  type        = number
  default     = 10

  validation {
    condition     = var.timeout >= 1 && var.timeout <= 900
    error_message = "timeout must be between 1 and 900 seconds."
  }
}

variable "log_retention_days" {
  description = "CloudWatch Logs retention in days."
  type        = number
  default     = 30

  validation {
    condition = contains(
      [1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1827, 2192, 2557, 2922, 3288, 3653, 0],
      var.log_retention_days,
    )
    error_message = "log_retention_days must be one of the values accepted by CloudWatch Logs, or 0 for never expire."
  }
}

variable "tags" {
  description = "Tags applied to all resources created by this module."
  type        = map(string)
  default     = {}
}

variable "kms_key_arn" {
  description = "Literal KMS key or alias ARN used by SSM to encrypt the private key parameter. Set only when the customer uses a CMK instead of the AWS-managed key. Null disables kms:Decrypt in the role policy."
  type        = string
  default     = null

  validation {
    condition = (
      var.kms_key_arn == null ||
      can(regex("^arn:aws[a-zA-Z-]*:kms:[a-z0-9-]+:[0-9]{12}:(key/[A-Za-z0-9-]+|alias/[A-Za-z0-9/_-]+)$", var.kms_key_arn))
    )
    error_message = "kms_key_arn must be a literal KMS key or alias ARN without wildcard characters."
  }
}
