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

##### - create an IAM-authenticated user on AWS DocumentDB
```hcl
# DocumentDB IAM auth: the "user" is the IAM user/role ARN, authenticated in the $external
# database via the MONGODB-AWS mechanism instead of a password - DocumentDB rejects a
# createUser/updateUser call that includes a password for this mechanism, so password must
# be left unset. See: https://docs.aws.amazon.com/documentdb/latest/developerguide/iam-identity-auth.html
resource "mongodb_db_user" "iam_user" {
  auth_database   = "$external"
  name            = "arn:aws:iam::123456789123:role/iamrole"
  auth_mechanisms = ["MONGODB-AWS"]
  role {
    role = "readWrite"
    db   = "my_database"
  }
}
```
## Argument Reference

* `auth_database` - (Required, Forces new resource) Database against which Mongo authenticates the user. A user must provide both a username and authentication database to log into MongoDB. MongoDB has no command to move an existing user to a different authentication database, so changing this destroys and recreates the resource. For IAM-authenticated DocumentDB users this must be `"$external"`.
* `role` - (optional) List of user’s roles and the databases / collections on which the roles apply. A role allows the user to perform particular actions on the specified database. A role on the admin database can include privileges that apply to the other databases as well. See [Role](#role) below for more details.

* `name` - (Required, Forces new resource) Username for authenticating to MongoDB. MongoDB has no user-rename command, so changing this destroys and recreates the resource. For IAM-authenticated DocumentDB users this is the IAM user/role ARN.
* `password` - (Optional) User's initial password. Required for normal SCRAM-authenticated users; must be left unset for a user authenticated via an external-identity mechanism (`auth_mechanisms`, e.g. `["MONGODB-AWS"]` for DocumentDB IAM auth, or MongoDB's own `MONGODB-X509`) — MongoDB rejects a `createUser`/`updateUser` call that includes a password for those. May be removed from your Terraform configuration after user creation without impacting the user, password or Terraform management. Updating this (or `role`) updates the existing user in place via MongoDB's `updateUser` command; it does not drop and recreate the user.
* `auth_mechanisms` - (Optional) Set of authentication mechanisms for this user, e.g. `["MONGODB-AWS"]` for an IAM-authenticated user on AWS DocumentDB. Leave unset for a normal password-authenticated (SCRAM) user.

~> **IMPORTANT:** --- Passwords may show up in Terraform related logs and it will be stored in the Terraform state file as plain-text. Password can be changed after creation using your preferred method, e.g. via the MongoDB Shell, to ensure security.  If you do change management of the password to outside of Terraform be sure to remove the argument from the Terraform configuration so it is not inadvertently updated to the original password.

### Role

Block mapping a user's role to a database / collection. A role allows the user to perform particular actions on the specified database. A role on the admin database can include privileges that apply to the other databases as well.

-> **NOTE:** The available privilege actions for custom MongoDB roles support a subset of MongoDB commands.

* `role` - (Required) Name of the role to grant. See [Create a Database User](https://docs.mongodb.com/manual/reference/method/db.createUser/#create-administrative-user-with-roles) `roles`.

-> **NOTE:** you can also use [built-in-roles](https://docs.mongodb.com/manual/reference/built-in-roles/index.html) 
* `db`   - (Required) Database on which the user has the specified role. A role on the `admin` database can include privileges that apply to the other databases.



## Import

Mongodb users can be imported using `<auth_database>/<name>`, e.g. for a user named `user_test` in database `test_db`:

```sh
$ terraform import mongodb_db_user.example_user test_db/user_test
```

-> **NOTE:** Prior to `v1` of this resource's state schema, the ID was `base64("<auth_database>.<name>")`. Existing state is migrated to the plain `<auth_database>/<name>` form automatically on the first `plan`/`apply` after upgrading — no manual `terraform state` changes are needed.