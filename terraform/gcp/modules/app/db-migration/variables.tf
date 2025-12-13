variable "project_id" { type = string }
variable "project_region" { type = string }
variable "network_name" { type = string }
variable "subnetwork_name" { type = string }

variable "db_secrets" {
  type = object({
    db_host     = string
    db_name     = string
    db_password = string
    db_user     = string
  })
}

variable "cloudrun_settings" { type = any }
