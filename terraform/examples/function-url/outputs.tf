output "function_url" {
  description = "HTTPS endpoint for the Lambda Function URL."
  value       = module.broker.function_url
}

output "function_arn" {
  description = "ARN of the Lambda function."
  value       = module.broker.function_arn
}
