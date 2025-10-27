variable "region" {
  type = string

  description = "Aws region to provision the infrastracture"

}

variable "bucket" {
  type        = string
  description = "S3 bucket for terraform state."
}

variable "github_repos" {
  type        = list(string)
  description = "GitHub repositories."
}
