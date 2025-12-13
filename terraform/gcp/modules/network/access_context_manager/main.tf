# Service Controls + Access Context Manager
resource "google_access_context_manager_access_level" "allow_access_level" {
  parent = "accessPolicies/000000"
  name   = "accessPolicies/000000/accessLevels/${var.environment}_allow_list"
  title  = "${var.environment}-allow-list"

  basic {
    combining_function = "OR"

    conditions {
      members = var.acm_settings.allow_sas
    }

    conditions {
      ip_subnetworks = var.acm_settings.company_office_ips
    }
  }
}

resource "google_access_context_manager_service_perimeter" "service_perimeter" {
  parent = "accessPolicies/000000"
  name   = "accessPolicies/000000/servicePerimeters/${var.environment}_service_controls"
  title  = "${var.environment}_service_controls"

  status {
    resources = [
      "projects/${var.project_number}"
    ]

    restricted_services = [
      "storage.googleapis.com"
    ]

    access_levels = [
      google_access_context_manager_access_level.allow_access_level.name
    ]
  }
}
