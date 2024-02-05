# Mongo Database Collection

Provides a Database Collection resource.

## Example Usages

##### - create collection
```hcl

resource "mongodb_db_collection" "collection_1" {
  db = "my_database"
  name = "example"
  deletion_protection = true
}
```

```
## Argument Reference
* `db`   - (Required) Database in which the colleciton will be created
* `name` - (Required) Collection name
* `keys` - (Required) Collection name
* `deletion_protection` - (Optional) Timeout for index creation operation. Default is 30 seconds


## Import

Mongodb collections can be imported using the hex encoded id, e.g. for a collection named `collection_test` and his database id `test_db` :

```sh
$ printf '%s' "test_db.collection_test" | base64
## this is the output of the command above it will encode db.collection to HEX 
dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3Q=

$ terraform import mongodb_db_collection.example_collection  dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3Q=
```