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
  host = "documentdb-test-terraform.cluster-ro-ctclcdufsrkx.eu-west-3.docdb.amazonaws.com"
  port = "27017"
  username = ""
  password = ""
  ssl = true
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
# https://docs.aws.amazon.com/documentdb/latest/developerguide/iam-identity-auth.html#iam-identity-auth-get-started
resource "mongodb_db_user" "passwordless_user" {
  auth_database = var.auth_database
  auth_mechanisms = var.auth_mechanisms
  name = "arn:aws:iam::123456789123:user/iamuser"
  role {
    role = "readAnyDatabase"
    db =   "test"
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
