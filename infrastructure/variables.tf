variable "subscription_id" {
  type        = string
  description = "Subscription ID for Azure"
}

variable "client_id" {
  type        = string
  sensitive   = true
  description = "Client ID for Service Principal to authenticate Terraform"
}

variable "client_secret" {
  type        = string
  sensitive   = true
  description = "Client secret for Service Principal to authenticate Terraform"
}

variable "tenant_id" {
  type        = string
  sensitive   = true
  description = "Tenant ID for Service Principal to authenticate Terraform"
}

variable "location" {
  type        = string
  default     = "eastus"
  description = "Location of the resource group"
}

variable "project" {
  type        = string
  description = "Name of project (all lowercase and no special characters)"
}

variable "environment" {
  type        = string
  default     = "dev"
  description = "The application environment"
}

variable "web_server_diag_account_tier" {
  type        = string
  default     = "Standard"
  description = "The storage account tier for the VM boot diagnostics"
}

variable "web_server_computer_name" {
  type        = string
  default     = "gowebserver"
  description = "The computer name for the virtual machine (not the resource name)"
}

variable "web_server_username" {
  type        = string
  default     = "azureuser"
  description = "The username for the virtual machine"
}

variable "web_server_diag_replication_type" {
  type        = string
  default     = "LRS"
  description = "The storage account replication type for the VM boot diagnostics"
}

variable "web_server_vm_size_sku" {
  type        = string
  default     = "Standard_B1ls"
  description = "The SKU of the virtual machine size"
}

variable "web_server_source_image_publisher" {
  type        = string
  default     = "Canonical"
  description = "The VM image publisher"
}

variable "web_server_source_image_offer" {
  type        = string
  default     = "0001-com-ubuntu-server-jammy"
  description = "The VM image OS"
}

variable "web_server_source_image_sku" {
  type        = string
  default     = "22_04-lts-gen2"
  description = "The VM image OS version"
}

variable "key_vault_sku_name" {
  type    = string
  default = "standard"
}

variable "key_permissions" {
  type        = list(string)
  description = "List of key permissions."
  default     = ["List", "Create", "Delete", "Get", "Purge", "Recover", "Update", "GetRotationPolicy", "SetRotationPolicy"]
}

variable "secret_permissions" {
  type        = list(string)
  description = "List of secret permissions."
  default     = ["Set", "Get", "Delete"]
}

variable "msi_id" {
  type        = string
  description = "The Managed Service Identity ID. If this value isn't null (the default), 'data.azurerm_client_config.current.object_id' will be set to this value."
  default     = null
}

variable "user_principal_name" {
  type        = string
  description = "The Entra user to give access to key vault"
}

variable "audio_storage_account_tier" {
  type        = string
  default     = "Standard"
  description = "The storage account tier for audio files"
}

variable "audio_storage_replication_type" {
  type        = string
  default     = "LRS"
  description = "The storage account replication type for audio files"
}

variable "openai_api_key" {
  type        = string
  description = "API key for using the Whisper and ChatGPT models"
  sensitive   = true
}

variable "notion_client_id" {
  type        = string
  description = "Client ID for public notion integration"
  sensitive   = true
}

variable "notion_client_secret" {
  type        = string
  description = "Client secret for public notion integration"
  sensitive   = true
}
