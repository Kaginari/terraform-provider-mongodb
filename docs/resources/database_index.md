# mongodb_db_index

`mongodb_db_index` provides a MongoDB index resource — lets you manage indexes on a collection through Terraform.

## Example Usage

```hcl
resource "mongodb_db_index" "email_unique" {
  db         = "app"
  collection = "users"

  keys {
    field = "email"
    value = "1"
  }
}
```

## Example Usage with a compound key, a partial filter, and a custom name

```hcl
resource "mongodb_db_index" "active_users_by_signup" {
  db         = "app"
  collection = "users"
  name       = "active_users_by_signup"

  keys {
    field = "signup_date"
    value = "-1"
  }
  keys {
    field = "email"
    value = "1"
  }

  partial_filter_expression = "{\"active\":{\"$eq\":true}}"
}
```

## Argument Reference

* `db` - (Required, Forces new resource) The database the collection lives in.
* `collection` - (Required, Forces new resource) The collection to index.
* `name` - (Optional, Computed, Forces new resource) Index name. If omitted, MongoDB's own default naming convention (each key field/direction joined by underscores, e.g. `email_1`) is used and read back into state.
* `keys` - (Required, Forces new resource) One or more blocks describing the index key spec, **in order** — order matters for compound indexes. See [Keys](#keys) below.
* `partial_filter_expression` - (Optional, Computed, Forces new resource) A JSON string for a [partial index](https://www.mongodb.com/docs/manual/core/index-partial/)'s filter expression, e.g. `"{\"active\":{\"$eq\":true}}"`.
* `hidden` - (Optional, Computed) `default = false`. [Hides the index](https://www.mongodb.com/docs/manual/core/index-hidden/) from the query planner without dropping it (MongoDB 4.4+). Updatable in place — toggling it does not rebuild the index.
* `timeout` - (Optional, Computed) `default = 30`. Seconds to allow the index build to run for.

### Keys

Each `keys` block describes one field in the index:

* `field` - (Required) The field name to index.
* `value` - (Required) `1`/`-1` for ascending/descending, or a special index type such as `"text"`/`"2dsphere"`.

## Import

Indexes can be imported using `<db>/<collection>/<name>`:

```sh
$ terraform import mongodb_db_index.email_unique app/users/email_1
```
