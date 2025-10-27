output "iam_oidc_provider_arn" {
  description = "ARN of the IAM OIDC provider"
  value       = module.iam_oidc_provider.arn
}
output "iam_role_arn" {
  description = "ARN of the IAM role for GitHub Actions"
  value       = module.iam_role.arn
}
