variable "project_region" { type = string }
variable "web_client_domain" { type = string }
variable "web_client_bucket_name" { type = string }
variable "api_server_service_name" { type = string }

variable "allowed_ip_ranges" {
  description = "List of IP ranges allowed to access the load balancer (CIDR notation)"
  type        = list(string)
  default     = []
}
