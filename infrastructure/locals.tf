locals {
  common_tags = {
    project             = "${var.project}"
    provisioning_method = "terraform"
    environment         = "${var.environment}"
  }

  current_user_id = coalesce(var.msi_id, data.azurerm_client_config.current.object_id)
}
