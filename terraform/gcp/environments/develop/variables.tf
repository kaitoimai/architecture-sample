# Project Configuration
variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "project_region" {
  description = "GCP Region for resources"
  type        = string
}

variable "project_number" {
  description = "GCP Project Number"
  type        = string
}

# Environment
variable "environment" {
  description = "Environment name (develop, staging, production)"
  type        = string
  validation {
    condition     = contains(["develop", "staging", "production"], var.environment)
    error_message = "Environment must be develop, staging, or production."
  }
}

# Domain
variable "web_client_domain" {
  description = "Domain for web client application"
  type        = string
}

# GitHub
variable "github_organization" {
  description = "GitHub organization name for repository"
  type        = string
}

# Monitoring
variable "monitoring_service_account" {
  description = "Service account email for external monitoring service"
  type        = string
}

# API Server Configuration
variable "api_server" {
  description = "API Server configuration"
  type = object({
    cpu                             = string
    memory                          = string
    max_scale                       = number
    min_scale                       = number
    topic                           = string
    cloud_tasks_data_import_job_url = string
    csv_upload_bucket               = string
  })
}

# Async Worker Configuration
variable "async_worker" {
  description = "Async Worker configuration"
  type = object({
    cpu       = string
    memory    = string
    max_scale = number
    min_scale = number
  })
}

# Database Migration Configuration
variable "db_migration" {
  description = "Database Migration job configuration"
  type = object({
    cpu         = string
    memory      = string
    max_retries = number
    timeout     = string
  })
}

# Cloud Build Configuration
variable "cloud_build" {
  description = "Cloud Build trigger configuration"
  type = object({
    api_server = object({
      trigger_name         = string
      deploy_target_branch = string
      build_file_path      = string
      push_tag             = string
      is_approval_config   = bool
      platform             = string
      trigger_id           = string
    })
    web_client = object({
      trigger_name       = string
      push_tag           = string
      is_approval_config = bool
    })
  })
}

# Monitoring APM Configuration
variable "monitoring_apm" {
  description = "Monitoring APM configuration"
  type = object({
    app_name       = string
    enabled_string = string
  })
}

# Access Context Manager Configuration
variable "acm" {
  description = "Access Context Manager configuration"
  type = object({
    company_office_ips = list(string)
  })
}

# Bastion Configuration
variable "bastion" {
  description = "Bastion host configuration"
  type = object({
    compute_address_name = string
    machine_type         = string
    zone                 = string
  })
}

# Monitoring Configuration
variable "monitoring" {
  description = "Cloud Monitoring configuration"
  type = object({
    enabled_alert           = bool
    notification_channel_id = string
  })
}
