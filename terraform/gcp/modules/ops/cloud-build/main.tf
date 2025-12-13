resource "google_cloudbuild_trigger" "api_server_deploy" {
  name               = var.cloudbuild_settings.api_server.trigger_name
  location           = var.project_region
  filename           = "cloudbuild.yaml"
  include_build_logs = "INCLUDE_BUILD_LOGS_WITH_STATUS"

  tags = [
    "gcp-cloud-build-deploy-cloud-run",
    "gcp-cloud-build-deploy-cloud-run-managed",
    "api-server"
  ]

  substitutions = {
    _SERVICE_NAME  = "api-server"
    _AR_HOSTNAME   = "asia-northeast1-docker.pkg.dev"
    _DEPLOY_REGION = var.project_region
    _PLATFORM      = var.cloudbuild_settings.api_server.platform
    _TRIGGER_ID    = var.cloudbuild_settings.api_server.trigger_id
  }

  github {
    name  = "api-server"
    owner = var.github_organization

    # Tag push trigger (for production deployments)
    dynamic "push" {
      for_each = var.cloudbuild_settings.api_server.push_tag != null ? [1] : []
      content {
        tag          = var.cloudbuild_settings.api_server.push_tag
        invert_regex = false
      }
    }

    # Branch push trigger (for non-production deployments)
    dynamic "push" {
      for_each = var.cloudbuild_settings.api_server.deploy_target_branch != null ? [1] : []
      content {
        branch       = var.cloudbuild_settings.api_server.deploy_target_branch
        invert_regex = false
      }
    }
  }

  approval_config {
    approval_required = var.cloudbuild_settings.api_server.is_approval_config
  }
}

resource "google_cloudbuild_trigger" "web_client_deploy" {
  name               = var.cloudbuild_settings.web_client.trigger_name
  location           = var.project_region
  filename           = "cloudbuild.yaml"
  include_build_logs = "INCLUDE_BUILD_LOGS_WITH_STATUS"

  tags = [
    "gcp-cloud-build-deploy-storage",
    "web-client"
  ]

  github {
    name  = "web-client"
    owner = var.github_organization

    # Tag push trigger
    dynamic "push" {
      for_each = var.cloudbuild_settings.web_client.push_tag != null ? [1] : []
      content {
        tag          = var.cloudbuild_settings.web_client.push_tag
        invert_regex = false
      }
    }
  }

  approval_config {
    approval_required = var.cloudbuild_settings.web_client.is_approval_config
  }
}
