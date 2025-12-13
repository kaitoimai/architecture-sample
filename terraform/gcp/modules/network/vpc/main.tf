resource "google_compute_network" "main" {
  name                     = "main-vpc"
  auto_create_subnetworks  = false
  enable_ula_internal_ipv6 = false
}

resource "google_compute_subnetwork" "main_subnetwork" {
  name          = "main-subnetwork"
  region        = var.project_region
  network       = google_compute_network.main.id
  ip_cidr_range = "10.0.1.0/24"
  stack_type    = "IPV4_ONLY"
}

# for alloydb
resource "google_compute_global_address" "alloydb_range" {
  name          = "alloy-db-range"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  address       = "10.1.0.0"
  network       = google_compute_network.main.id
}

resource "google_service_networking_connection" "alloydb_private_connection" {
  network                 = google_compute_network.main.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.alloydb_range.name]
}
