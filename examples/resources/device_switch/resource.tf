terraform {
  required_providers {
    unifi = {
      source = "jamestoyer/unifi"
    }
  }
}

provider "unifi" {
  insecure = true
  url      = "https://127.0.0.1:8443"
  username = "admin"
  password = "admin"
}

resource "unifi_device_switch" "example" {
  name = "Example Switch"
  mac  = "00:27:22:00:00:05"

  management_network_id = "66a5357b30079358c34fe5d9"

  static_ip_settings = {
    ip            = "10.2.3.4"
    gateway       = "10.2.0.1"
    netmask       = "255.255.255.0"
    preferred_dns = "1.2.3.4"
  }

  port_overrides = {
    "1" = {
      name     = "Disabled"
      disabled = true
    }
    "2" = {
      excluded_tagged_network_ids = [
        "66a52a6c30079358c34f3151",
      ]
      name                   = "Custom Tagged VLAN"
      native_network_id      = "66a5358030079358c34fe5db"
      operation              = "switch"
      poe_mode               = "auto"
      tagged_vlan_management = "custom"
    },
    "3" = {
      name                   = "Block All Tagged VLAN"
      tagged_vlan_management = "block_all"
    }
    "4" = {
      name              = "Native Network Override"
      native_network_id = "66a52a6c30079358c34f3151"
    }
    "5" = {
      full_duplex = true
      link_speed  = "1000"
      name        = "Link Speed"
      operation   = "switch"
    }
    "6" = {
      poe_mode = "off"
      name     = "POE Off"
    }
    "7" = {
      name = "Named Port"
    }
    "8" = {
      port_profile_id = "669c1ef8329aae15c4b3f791"
      name            = "Port Profile"
    }
    "9" = {
      name              = "Disabled native network"
      native_network_id = ""
    }
    "10" = {
      aggregate_num_ports = 2
      operation           = "aggregate"
      name                = "Aggregate 1"
    }
    "11" = {
      name = "Aggregate 2"
    }
    "12" = {
      name = "Mirror Target"
    }
    "13" = {
      mirror_port_index = 11
      name              = "Mirror Root"
      operation         = "mirror"
    }
  }
}
