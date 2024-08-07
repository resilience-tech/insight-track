provider "azurerm" {
  features {}
}

variable "main_resource_group_name" {
  default = "synthetic-scan-web-rg"
}

variable "servicebus_resource_group_name" {
  default = "observability-rg"
}

variable "location" {
  default = "WestEurope"
}

variable "servicebus_namespace" {
  default = "observability-message-ns"
}

variable "input_topic_name" {
  default = "synthetic-scan-request"
}

variable "output_topic_name" {
  default = "synthetic-scan-result"
}

variable "function_app_name" {
  default = "synthetic-scan-web-fa"
}

variable "storage_account_name" {
  default = "syscanwebfas"
}

# Create the Service Bus resources in a separate resource group
resource "azurerm_resource_group" "servicebus_rg" {
  name     = var.servicebus_resource_group_name
  location = var.location
}

resource "azurerm_servicebus_namespace" "sb_namespace" {
  name                = var.servicebus_namespace
  location            = azurerm_resource_group.servicebus_rg.location
  resource_group_name = azurerm_resource_group.servicebus_rg.name
  sku                 = "Standard"
}

# Use the synthetic_scan module for the main resource group
module "synthetic_scan" {
  source                        = "../synthetic-scan-ic-tf/modules"
  main_resource_group_name      = var.main_resource_group_name
  location                      = var.location
  storage_account_name          = var.storage_account_name
  function_app_name             = var.function_app_name
  service_bus_connection_string = azurerm_servicebus_namespace.sb_namespace.default_primary_connection_string
}

resource "azurerm_role_assignment" "servicebus_role" {
  principal_id         = module.synthetic_scan.function_app_identity_principal_id
  role_definition_name = "Azure Service Bus Data Owner"
  scope                = azurerm_servicebus_namespace.sb_namespace.id
}
