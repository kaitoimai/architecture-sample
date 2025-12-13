output "cluster_name" {
  value = google_alloydb_cluster.main_cluster.name
}

output "instance_name" {
  value = google_alloydb_instance.main_instance.name
}
