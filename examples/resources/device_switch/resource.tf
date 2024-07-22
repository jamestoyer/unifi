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

data "unifi_device_switch" "example" {
  mac = "00:27:22:00:00:01"
}

import {
  id = data.unifi_device_switch.example.id
  to = unifi_device_switch.example
}

resource "unifi_device_switch" "example" {
  name = "Example Switch Updates"
  mac  = data.unifi_device_switch.example.mac

  management_network_id = "669c0336329aae15c4b318f2"

  static_ip_settings = {
    ip            = "10.2.3.4"
    gateway       = "10.2.0.1"
    netmask       = "255.255.255.0"
    preferred_dns = "1.2.3.4"
  }

  port_overrides = {
    "38" = {
      name              = "Native Network Override"
      native_network_id = "669c08b2329aae15c4b3d60a"
    }
    "39" = {
      full_duplex = true
      link_speed  = "1000"
      name        = "Link Speed"
      operation   = "switch"
    }
    "40" = {
      poe_mode = "off"
      name     = "POE Off"
    }
    "41" = {
      name = "Named Port"
    }
    #       "42" = {
    #         port_profile_id = "669c1ef8329aae15c4b3f791"
    #         name = "Port Profile ID"
    #       }
    "44" = {
      name              = "Disabled native network"
      native_network_id = ""
    }
    "45" = {
      aggregate_num_ports = 2
      operation           = "aggregate"
      name                = "Aggregate 1"
    }
    "46" = {
      name = "Aggregate 2"
    }
    "47" = {
      #         disabled = true
      name = "Mirror Target"
    }
    "48" = {
      #         disabled = true
      mirror_port_index = 47
      name              = "Mirror Root"
      operation         = "mirror"
    }
  }
}
