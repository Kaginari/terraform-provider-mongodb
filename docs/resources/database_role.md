# mongodb_db_role

`mongodb_db_role` provides a Custom DB Role resource. The customDBRoles resource lets you retrieve, create and modify the custom MongoDB roles in your mongo database server. Use custom MongoDB roles to specify custom sets of privileges.


## Example Usages

```hcl
resource "mongodb_db_role" "example_role" {
  name = "role_name"
  database = "my_database"
  privilege {
    db = "admin"
    collection = "*"
    actions = ["collStats"]
  }
  privilege {
    db = "my_database"
    collection = ""
    actions = ["listCollections", "createCollection","createIndex", "dropIndex", "insert", "remove", "renameCollectionSameDB", "update"]
  }


}
```
## Example Usage with inherited roles

```hcl
resource "mongodb_db_role" "role" {
  database = "admin"
  name = "new_role"
  privilege {
    db = "admin"
    collection = ""
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
}
```
## Argument Reference

* `database` - (Optional) **default="admin"** The database of the role.

~> **IMPORTANT:** If a role is created in a specific database you can only use it as inherited in another role in the same database.

* `name` - (Required) Name of the custom role.

	-> **NOTE:** The specified role name can only contain letters, digits, underscores, and dashes. Additionally, you cannot specify a role name which meets any of the following criteria:

	* Is a name already used by an existing custom role
	* Is a name of any of the built-in roles see [built-in-roles](https://docs.mongodb.com/manual/reference/built-in-roles/index.html)

### Privilege
Each object in the privilege array represents an individual privilege action granted by the role. It is not required.

* `actions` - (Required) Array of the privilege action. For a complete list of actions available , see [Custom Role Actions](https://docs.mongodb.com/manual/reference/privilege-actions/)
-> **Note**: The privilege actions available to the Custom Roles API resource represent a subset of the privilege actions available in the Atlas Custom Roles UI.
* `db`	Database on which the action is granted.
* `collection` - (Optional) Collection on which the action is granted. 
-> **Note**: If collection value is an empty string, the actions are granted on all collections within the database specified in the privilege.db field.
             
### Inherited Roles
Each object in the inheritedRoles array represents a key-value pair indicating the inherited role and the database on which the role is granted. It is an optional field.

* `db` (Required) Database on which the inherited role is granted.

	-> **NOTE** This value should be admin for all roles except read and readWrite.

* `role`	(Required) Name of the inherited role. This can either be another custom role or a [built-in role](https://docs.mongodb.com/manual/reference/built-in-roles/index.html).


## Import

## Import

Mongodb users can be imported using the hex encoded id, e.g. for a user named `user_test` and his database id `test_db` :

```sh
$ printf '%s' "test_db.role_test"  | base64
## this is the output of the command above it will encode db.rolename to HEX 
dGVzdF9kYi5yb2xlX3Rlc3Q=

$ terraform import mongodb_db_role.example_role  dGVzdF9kYi5yb2xlX3Rlc3Q=
```