resource "azurerm_resource_group" "project_rg" {
  name     = "rg-${var.project}-${var.environment}"
  location = var.location

  tags = local.common_tags
}

resource "azapi_resource_action" "ssh_public_key_gen" {
  type        = "Microsoft.Compute/sshPublicKeys@2022-11-01"
  resource_id = azapi_resource.ssh_public_key.id
  action      = "generateKeyPair"
  method      = "POST"

  response_export_values = ["publicKey", "privateKey"]
}

resource "azapi_resource" "ssh_public_key" {
  type      = "Microsoft.Compute/sshPublicKeys@2022-11-01"
  name      = "sshkey-${var.project}-${var.environment}"
  location  = azurerm_resource_group.project_rg.location
  parent_id = azurerm_resource_group.project_rg.id
}

resource "azurerm_virtual_network" "web_server_vnet" {
  name                = "vnet-${var.project}-${var.environment}"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.project_rg.location
  resource_group_name = azurerm_resource_group.project_rg.name

  tags = local.common_tags
}

resource "azurerm_subnet" "web_server_snet" {
  name                 = "snet-${var.project}-${var.environment}"
  resource_group_name  = azurerm_resource_group.project_rg.name
  virtual_network_name = azurerm_virtual_network.web_server_vnet.name
  address_prefixes     = ["10.0.0.0/24"]
}

resource "azurerm_public_ip" "web_server_pip" {
  name                = "pip-${var.project}-${var.environment}"
  location            = azurerm_resource_group.project_rg.location
  resource_group_name = azurerm_resource_group.project_rg.name
  allocation_method   = "Static"

  tags = local.common_tags
}


resource "azurerm_network_security_group" "web_server_nsg" {
  name                = "nsg-${var.project}-${var.environment}"
  location            = azurerm_resource_group.project_rg.location
  resource_group_name = azurerm_resource_group.project_rg.name

  tags = local.common_tags

  security_rule {
    name                       = "ssh"
    priority                   = 1001
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
  security_rule {
    name                       = "http"
    priority                   = 1002
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
  security_rule {
    name                       = "https"
    priority                   = 1003
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_network_interface" "web_server_nic" {
  name                = "nic-${var.project}-${var.environment}"
  location            = azurerm_resource_group.project_rg.location
  resource_group_name = azurerm_resource_group.project_rg.name

  tags = local.common_tags

  ip_configuration {
    name                          = "web_server_nic_configuration"
    subnet_id                     = azurerm_subnet.web_server_snet.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.web_server_pip.id
  }
}

resource "azurerm_network_interface_security_group_association" "web_server_security_group_association" {
  network_interface_id      = azurerm_network_interface.web_server_nic.id
  network_security_group_id = azurerm_network_security_group.web_server_nsg.id
}

resource "azurerm_storage_account" "web_server_diag_sa" {
  name                     = "diag${var.project}${var.environment}"
  location                 = azurerm_resource_group.project_rg.location
  resource_group_name      = azurerm_resource_group.project_rg.name
  account_tier             = var.web_server_diag_account_tier
  account_replication_type = var.web_server_diag_replication_type

  tags = local.common_tags
}

resource "azurerm_linux_virtual_machine" "web_server_vm" {
  name                  = "vm-${var.project}-${var.environment}"
  location              = azurerm_resource_group.project_rg.location
  resource_group_name   = azurerm_resource_group.project_rg.name
  network_interface_ids = [azurerm_network_interface.web_server_nic.id]
  size                  = var.web_server_vm_size_sku

  tags = local.common_tags

  os_disk {
    name                 = "osdisk-${var.project}-${var.environment}"
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = var.web_server_source_image_publisher
    offer     = var.web_server_source_image_offer
    sku       = var.web_server_source_image_sku
    version   = "latest"
  }

  computer_name  = var.web_server_computer_name
  admin_username = var.web_server_username

  admin_ssh_key {
    username   = var.web_server_username
    public_key = azapi_resource_action.ssh_public_key_gen.output.publicKey
  }

  boot_diagnostics {
    storage_account_uri = azurerm_storage_account.web_server_diag_sa.primary_blob_endpoint
  }
}

# Key Vault
data "azurerm_client_config" "current" {}

data "azuread_user" "entra_user" {
  user_principal_name = var.user_principal_name
}

resource "azurerm_key_vault" "web_server_key_vault" {
  name                       = "kv-${var.project}-${var.environment}"
  location                   = azurerm_resource_group.project_rg.location
  resource_group_name        = azurerm_resource_group.project_rg.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = var.key_vault_sku_name
  soft_delete_retention_days = 7
}

resource "azurerm_key_vault_access_policy" "service_principal_access_policy" {
  key_vault_id = azurerm_key_vault.web_server_key_vault.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = local.current_user_id

  key_permissions    = var.key_permissions
  secret_permissions = var.secret_permissions
}

resource "azurerm_key_vault_access_policy" "entra_user_access_policy" {
  key_vault_id = azurerm_key_vault.web_server_key_vault.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = data.azuread_user.entra_user.object_id

  key_permissions    = var.key_permissions
  secret_permissions = var.secret_permissions
}

resource "azurerm_key_vault_key" "web_server_key" {
  name = "key-test"

  key_vault_id = azurerm_key_vault.web_server_key_vault.id
  key_type     = var.key_type
  key_size     = var.key_size
  key_opts     = var.key_ops

  rotation_policy {
    automatic {
      time_before_expiry = "P30D"
    }

    expire_after         = "P90D"
    notify_before_expiry = "P29D"
  }
}
