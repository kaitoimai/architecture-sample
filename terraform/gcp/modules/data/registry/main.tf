resource "google_artifact_registry_repository" "docker" {
  location      = var.project_region
  repository_id = "docker"
  description   = "docker repository"
  format        = "DOCKER"
}
