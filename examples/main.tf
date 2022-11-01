terraform {
  required_version = ">= 0.13"

  required_providers {
    mongodb = {
      source = "registry.terraform.io/Kaginari/mongodb"
      version = "9.9.9"
    }
  }
}

provider "mongodb" {
  host = "mongo"
  port = "27017"
  username = "root"
  password = "root"
  ssl = false
  auth_database = "admin"
  proxy = "socks5://localhost:1080"
}

variable "username" {
  description = "the user name"
  default = "monta"
}
variable "password" {
  description = "the user password"
  default = "monta"
}

resource "mongodb_db_role" "role" {
  name = "custom_role_test"
  privilege {
    db = "admin"
    collection = "*"
    actions = ["collStats"]
  }
  privilege {
    db = "ds"
    collection = "*"
    actions = ["collStats"]
  }


}

resource "mongodb_db_role" "role_2" {
  depends_on = [mongodb_db_role.role]
  database = "admin"
  name = "new_role3"
  inherited_role {
    role = mongodb_db_role.role.name
    db =   "admin"
  }
  privilege {
    db = "not_inhireted"
    collection = "*"
    actions = ["collStats"]
  }
}
resource "mongodb_db_role" "role4" {
  depends_on = [mongodb_db_role.role]
  database = "exemple"
  name = "new_role4"
}

resource "mongodb_db_user" "user" {
  auth_database = "exemple"
  name = "monta"
  password = "monta"
  role {
    role = mongodb_db_role.role.name
    db =   "admin"
  }
  role {
    role = "readAnyDatabase"
    db =   "admin"
  }
  role {
    role = "readWrite"
    db =   "local"
  }
  role {
    role = "readWrite"
    db =   "monta"
  }


}
