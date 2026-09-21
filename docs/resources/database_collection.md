# mongodb_db_collection

`mongodb_db_collection` provides a MongoDB collection resource — lets you create, configure and (carefully) delete collections through Terraform, rather than only the users/roles that access them.

## Example Usage

```hcl
resource "mongodb_db_collection" "events" {
  db   = "analytics"
  name = "events"

  # MongoDB 6.0+ - lets change streams observe the full pre/post document, not just the delta
  change_stream_pre_and_post_images = true
}
```

## Argument Reference

* `db` - (Required, Forces new resource) The database the collection lives in.
* `name` - (Required, Forces new resource) The collection name.
* `deletion_protection` - (Optional) `default = true`. Client-side only — never sent to MongoDB. When true, `Delete` refuses to drop the collection. Set it to `false` first if you actually want `terraform destroy` (or a plan that removes this resource) to drop it.
* `change_stream_pre_and_post_images` - (Optional) `default = false`. Enables pre- and post-images for change streams on this collection ([MongoDB 6.0+](https://www.mongodb.com/docs/manual/changeStreams/#change-streams-with-document-pre--and-post-images)). Updatable in place — toggling it does not recreate the collection.

## Import

Collections can be imported using `<db>/<name>`:

```sh
$ terraform import mongodb_db_collection.events analytics/events
```
