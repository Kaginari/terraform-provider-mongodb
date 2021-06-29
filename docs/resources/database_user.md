# Mongo Database User

Provides a Database User resource.

Each user has a set of roles that provide access to the databases.

~> **IMPORTANT:** All arguments including the password will be stored in the raw state as plain-text. [Read more about sensitive data in state.](https://www.terraform.io/docs/state/sensitive-data.html)

## Example Usages

##### - create user with predefined role
```hcl

resource "mongodb_db_user" "user" {
  auth_database = "my_database"
  name = "example"
  password = "example"
  role {
    role = "readAnyDatabase"
    db =   "my_database"
  }

}
```

##### - create user with [custom role]() `example_role`
```hcl
variable "username" {
  description = "the user name"
}
variable "password" {
  description = "the user password"
}

resource "mongodb_db_user" "user_with_custom role" {
  depends_on = [mongodb_db_role.example_role]
  auth_database = "my_database"
  name = var.username
  password = var.password
  role {
    role = mongodb_db_role.example_role.name
    db =   "my_database"
  }
  role {
    role = "readAnyDatabase"
    db =   "admin"
  }
}
```
## Argument Reference

* `auth_database` - (Required) Database against which Mongo authenticates the user. A user must provide both a username and authentication database to log into MongoDB.
* `role` - (optional) List of user’s roles and the databases / collections on which the roles apply. A role allows the user to perform particular actions on the specified database. A role on the admin database can include privileges that apply to the other databases as well. See [Role](#role) below for more details.

* `name` - (Required) Username for authenticating to MongoDB.
* `password` - (Required) User's initial password. A value is required to create the database user, however the argument but may be removed from your Terraform configuration after user creation without impacting the user, password or Terraform management. 

~> **IMPORTANT:** --- Passwords may show up in Terraform related logs and it will be stored in the Terraform state file as plain-text. Password can be changed after creation using your preferred method, e.g. via the MongoDB Shell, to ensure security.  If you do change management of the password to outside of Terraform be sure to remove the argument from the Terraform configuration so it is not inadvertently updated to the original password.

### Role

Block mapping a user's role to a database / collection. A role allows the user to perform particular actions on the specified database. A role on the admin database can include privileges that apply to the other databases as well.

-> **NOTE:** The available privilege actions for custom MongoDB roles support a subset of MongoDB commands.

* `role` - (Required) Name of the role to grant. See [Create a Database User](https://docs.mongodb.com/manual/reference/method/db.createUser/#create-administrative-user-with-roles) `roles`.

-> **NOTE:** you can also use [built-in-roles](https://docs.mongodb.com/manual/reference/built-in-roles/index.html) 
* `db`   - (Required) Database on which the user has the specified role. A role on the `admin` database can include privileges that apply to the other databases.



## Import

Mongodb users can be imported using the hex encoded id, e.g. for a user named `user_test` and his database id `test_db` :

```sh
$ printf '%s' "test_db.user_test" | base64
## this is the output of the command above it will encode db.username to HEX 
dGVzdF9kYi51c2VyX3Rlc3Q=

$ terraform import mongodb_db_user.example_user  dGVzdF9kYi51c2VyX3Rlc3Q=
```