resource "google_alloydb_cluster" "main_cluster" {
  cluster_id = "main-cluster"
  location   = var.project_region
  network_config {
    network = var.network_id
  }
  continuous_backup_config {
    enabled              = true
    recovery_window_days = 14
  }
}

resource "google_alloydb_instance" "main_instance" {
  instance_id   = "main-instance"
  cluster       = google_alloydb_cluster.main_cluster.name
  instance_type = "PRIMARY"

  database_flags = {
    "alloydb.logical_decoding"                            = "on"
    "alloydb.enable_pgaudit"                              = "on"
    "password.enforce_complexity"                         = "on"
    "password.enforce_expiration"                         = "on"
    "password.enforce_password_does_not_contain_username" = "on"
    "password.expiration_in_days"                         = "10000"
    "password.min_numerical_chars"                        = "1"
    "password.min_pass_length"                            = "10"
    "password.min_uppercase_letters"                      = "1"
    "password.notify_expiration_in_days"                  = "30"
    "pgaudit.log"                                         = "all"
  }

  depends_on = [var.alloydb_private_connection]
}
