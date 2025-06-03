variable "username" {
  description = "the user name"
  default = "monta_username"
}
variable "password" {
  description = "the user password"
  default = "monta_password"
}

variable "auth_database" {
  description = "Database against which DocumentDB authenticates the user."
  type = string
  default = "$external"
}

variable "auth_mechanisms" {
  description = "Auth mechanism."
  type = list(string)
  default = ["MONGODB-AWS"]
}
