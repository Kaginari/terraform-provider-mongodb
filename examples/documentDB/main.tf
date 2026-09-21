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

# IAM-authenticated user (AWS DocumentDB): the "user" is the IAM role/user ARN, no password -
# DocumentDB rejects createUser/updateUser if one is included for this auth mechanism.
# https://docs.aws.amazon.com/documentdb/latest/developerguide/iam-identity-auth.html
resource "mongodb_db_user" "iam_user" {
  auth_database   = "$external"
  name            = "arn:aws:iam::123456789123:role/iamrole"
  auth_mechanisms = ["MONGODB-AWS"]
  role {
    role = "readWrite"
    db   = "monta"
  }
}