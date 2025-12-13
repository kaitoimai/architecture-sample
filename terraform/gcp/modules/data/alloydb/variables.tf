variable "project_region" {
  type = string
}

variable "network_id" {
  description = "VPC network ID"
  type        = string
}

variable "alloydb_private_connection" {
  description = "Service networking connection"
  type        = any
}
