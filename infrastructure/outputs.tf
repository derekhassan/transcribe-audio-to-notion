output "public_key_data" {
  value = azapi_resource_action.ssh_public_key_gen.output.publicKey
}

output "private_key_data" {
  sensitive = true
  value     = azapi_resource_action.ssh_public_key_gen.output.privateKey
}

output "vm_admin_username" {
  value = azurerm_linux_virtual_machine.web_server_vm.admin_username
}

output "vm_ip_address" {
  value = azurerm_linux_virtual_machine.web_server_vm.public_ip_address
}
