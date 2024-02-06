terraform {
  required_version = ">= 0.13"

  required_providers {
    mongodb = {
      source = "registry.terraform.io/FelGel/mongodb"
      version = "9.9.9"
    }
  }
}
provider "mongodb" {
  host = "documentdb-test-terraform.cluster-ro-ctclcdufsrkx.eu-west-3.docdb.amazonaws.com"
  port = "27017"
  username = ""
  password = ""
  tls = true
  direct = true
  certificate = file(pathexpand("rds-combined-ca-bundle.pem"))
}
resource "mongodb_db_user" "user" {
  auth_database = "admin"
  name = "monta"
  password = "monta"
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