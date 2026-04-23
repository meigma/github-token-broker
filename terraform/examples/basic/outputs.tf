output "function_name" {
  description = "Name of the deployed Lambda function."
  value       = module.broker.function_name
}

output "function_arn" {
  description = "ARN of the deployed Lambda function."
  value       = module.broker.function_arn
}
