# Mongo Database Index

Provides a Database Index resource.

## Example Usages

##### - create index

```hcl

resource "mongodb_db_index" "collection_1" {
  db         = "my_database"
  collection = "example"
  name       = "my_index"
  keys {
    field = "field_name_to_index2"
    value = "-1"
  }
  keys {
    field = "field_name_to_index"
    value = "1"
  }
  timeout = 30
}
```

## Argument Reference
* `db` - (Required) Database in which the target colleciton resides
* `collection` - (Required) Collection name
* `keys` - (Required) Field and value pairs where the field is the index key and the value describes the type of index for that field
                      For an ascending index on a field, specify a value of 1. For descending index, specify a value of -1
                      See https://www.mongodb.com/docs/manual/reference/method/db.collection.createIndex/ for details
* `name` - (Optional) Index name
* `timeout` - (Optional) Timeout for index creation operation


## Import

Mongodb indexes can be imported using the hex encoded id, e.g. for a collection named `collection_test`, his database id `test_db` and collection name `example_index`:

```sh
$ printf '%s' "test_db.collection_test.example_index" | base64
## this is the output of the command above it will encode db.collection.index to HEX 
dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3QuZXhhbXBsZV9pbmRleA==

$ terraform import mongodb_db_index.example_index  dGVzdF9kYi5jb2xsZWN0aW9uX3Rlc3QuZXhhbXBsZV9pbmRleA==
```