output "function_arn" {
  description = "ARN of the Lambda function."
  value       = aws_lambda_function.broker.arn
}

output "function_name" {
  description = "Name of the Lambda function."
  value       = aws_lambda_function.broker.function_name
}

output "function_invoke_arn" {
  description = "Invoke ARN, suitable for API Gateway or EventBridge integrations."
  value       = aws_lambda_function.broker.invoke_arn
}

output "role_arn" {
  description = "ARN of the Lambda execution role."
  value       = aws_iam_role.lambda.arn
}

output "role_name" {
  description = "Name of the Lambda execution role."
  value       = aws_iam_role.lambda.name
}

output "log_group_name" {
  description = "Name of the CloudWatch Log Group backing Lambda logs."
  value       = aws_cloudwatch_log_group.lambda.name
}

output "deployed_version" {
  description = "Release version actually deployed, or null when the module was pointed at a local zip or S3 source."
  value       = try(var.lambda_artifact.release_version, null)
}
