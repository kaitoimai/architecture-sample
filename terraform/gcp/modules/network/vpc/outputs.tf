output "network_id" {
  description = "VPC network ID (self_link)"
  value       = google_compute_network.main.id
}

output "network_name" {
  description = "VPC network name"
  value       = google_compute_network.main.name
}

output "subnetwork_id" {
  description = "Subnet ID (self_link)"
  value       = google_compute_subnetwork.main_subnetwork.id
}

output "subnetwork_name" {
  description = "Subnet name"
  value       = google_compute_subnetwork.main_subnetwork.name
}

output "alloydb_private_connection" {
  description = "AlloyDB service networking connection"
  value       = google_service_networking_connection.alloydb_private_connection
}
