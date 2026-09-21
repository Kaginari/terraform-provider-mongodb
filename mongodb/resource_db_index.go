package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func resourceDatabaseIndex() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDatabaseIndexCreate,
		ReadContext:   resourceDatabaseIndexRead,
		UpdateContext: resourceDatabaseIndexUpdate,
		DeleteContext: resourceDatabaseIndexDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"db": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"collection": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Description: "Index name. If omitted, MongoDB's own default naming convention " +
					"(each key field/direction joined by underscores) is used and read back here.",
			},
			"keys": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MinItems: 1,
				Description: "The index key spec, in order (order matters for compound indexes - " +
					"this is a list, not a set).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field": {
							Type:     schema.TypeString,
							Required: true,
							// ForceNew on the containing "keys" list does not cascade to
							// its nested attributes in SDK v2 - a diff at "keys.N.field"
							// only forces replacement if this leaf itself is ForceNew.
							ForceNew: true,
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "1 or -1 for ascending/descending, or a special index type such as \"text\"/\"2dsphere\".",
						},
					},
				},
			},
			"partial_filter_expression": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "A JSON string for a partial index's filter expression, e.g. {\"field\":{\"$exists\":true}}.",
			},
			"hidden": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Hides the index from the query planner (MongoDB 4.4+). Updatable in place via collMod - does not require rebuilding the index.",
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Seconds to allow the index build to run for.",
			},
		},
	}
}

type indexKey struct {
	Field string `mapstructure:"field"`
	Value string `mapstructure:"value"`
}

func resourceDatabaseIndexCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	config := i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	database := data.Get("db").(string)
	collection := data.Get("collection").(string)

	keys, err := decodeIndexKeys(data.Get("keys").([]interface{}))
	if err != nil {
		return diag.Errorf("Error decoding keys : %s ", err)
	}

	name := data.Get("name").(string)
	if name == "" {
		name = defaultIndexName(keys)
	}

	indexSpec := bson.D{{Key: "key", Value: buildIndexKeyDocument(keys)}, {Key: "name", Value: name}}
	if pfe := data.Get("partial_filter_expression").(string); pfe != "" {
		var filter bson.M
		if err := bson.UnmarshalExtJSON([]byte(pfe), true, &filter); err != nil {
			return diag.Errorf("Error parsing partial_filter_expression as JSON : %s ", err)
		}
		indexSpec = append(indexSpec, bson.E{Key: "partialFilterExpression", Value: filter})
	}
	if data.Get("hidden").(bool) {
		indexSpec = append(indexSpec, bson.E{Key: "hidden", Value: true})
	}

	cmd := bson.D{
		{Key: "createIndexes", Value: collection},
		{Key: "indexes", Value: bson.A{indexSpec}},
	}
	if timeout, ok := data.GetOk("timeout"); ok {
		cmd = append(cmd, bson.E{Key: "maxTimeMS", Value: timeout.(int) * 1000})
	}

	result := client.Database(database).RunCommand(ctx, cmd)
	if result.Err() != nil {
		return diag.Errorf("Could not create the index : %s ", result.Err())
	}

	data.SetId(strings.Join([]string{database, collection, name}, "/"))
	return resourceDatabaseIndexRead(ctx, data, i)
}

func resourceDatabaseIndexRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	config := i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	stateID := data.State().ID
	database, collection, name, err := resourceDatabaseIndexParseId(stateID)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	idx, exists, err := getIndex(ctx, client, database, collection, name)
	if err != nil {
		return diag.Errorf("Error reading index : %s ", err)
	}
	if !exists {
		// Deleted out of band (or the collection/index was dropped some other way) -
		// treat as drift so Terraform offers to recreate it, rather than erroring forever.
		data.SetId("")
		return nil
	}

	if err := data.Set("db", database); err != nil {
		return diag.Errorf("error setting db : %s ", err)
	}
	if err := data.Set("collection", collection); err != nil {
		return diag.Errorf("error setting collection : %s ", err)
	}
	if err := data.Set("name", idx.name); err != nil {
		return diag.Errorf("error setting name : %s ", err)
	}
	if err := data.Set("keys", idx.keysForState()); err != nil {
		return diag.Errorf("error setting keys : %s ", err)
	}
	if err := data.Set("hidden", idx.hidden); err != nil {
		return diag.Errorf("error setting hidden : %s ", err)
	}
	if err := data.Set("partial_filter_expression", idx.partialFilterExpression); err != nil {
		return diag.Errorf("error setting partial_filter_expression : %s ", err)
	}

	data.SetId(stateID)
	return nil
}

func resourceDatabaseIndexUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	config := i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	database := data.Get("db").(string)
	collection := data.Get("collection").(string)
	name := data.Get("name").(string)

	// "db", "collection", "keys", "partial_filter_expression" and "name" are all
	// ForceNew - the only attribute that can reach here is "hidden".
	if data.HasChange("hidden") {
		hidden := data.Get("hidden").(bool)
		result := client.Database(database).RunCommand(ctx, bson.D{
			{Key: "collMod", Value: collection},
			{Key: "index", Value: bson.D{{Key: "name", Value: name}, {Key: "hidden", Value: hidden}}},
		})
		if result.Err() != nil {
			return diag.Errorf("Could not update hidden : %s ", result.Err())
		}
	}

	return resourceDatabaseIndexRead(ctx, data, i)
}

func resourceDatabaseIndexDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	config := i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	database, collection, name, err := resourceDatabaseIndexParseId(data.State().ID)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	result := client.Database(database).RunCommand(ctx, bson.D{
		{Key: "dropIndexes", Value: collection},
		{Key: "index", Value: name},
	})
	if result.Err() != nil {
		return diag.Errorf("Could not delete the index : %s ", result.Err())
	}
	return nil
}

func resourceDatabaseIndexParseId(id string) (string, string, string, error) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("unexpected format of ID (%s), expected database/collection/name", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func decodeIndexKeys(raw []interface{}) ([]indexKey, error) {
	keys := make([]indexKey, 0, len(raw))
	for _, r := range raw {
		m := r.(map[string]interface{})
		keys = append(keys, indexKey{Field: m["field"].(string), Value: m["value"].(string)})
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("at least one key is required")
	}
	return keys, nil
}

// indexKeyValue converts a key's string value into the BSON type MongoDB expects: a number
// for 1/-1 (direction), or the raw string for special index types like "text"/"2dsphere".
func indexKeyValue(value string) interface{} {
	if n, err := strconv.Atoi(value); err == nil {
		return n
	}
	return value
}

func buildIndexKeyDocument(keys []indexKey) bson.D {
	doc := make(bson.D, 0, len(keys))
	for _, k := range keys {
		doc = append(doc, bson.E{Key: k.Field, Value: indexKeyValue(k.Value)})
	}
	return doc
}

// defaultIndexName mirrors MongoDB's own default index-naming convention: each key
// field/value joined by underscores, e.g. {email: 1, created_at: -1} -> "email_1_created_at_-1".
func defaultIndexName(keys []indexKey) string {
	parts := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		parts = append(parts, k.Field, k.Value)
	}
	return strings.Join(parts, "_")
}

type mongoIndex struct {
	name                    string
	keys                    bson.D
	hidden                  bool
	partialFilterExpression string
}

func (idx mongoIndex) keysForState() []interface{} {
	out := make([]interface{}, 0, len(idx.keys))
	for _, e := range idx.keys {
		out = append(out, map[string]interface{}{
			"field": e.Key,
			"value": fmt.Sprintf("%v", e.Value),
		})
	}
	return out
}

type listIndexesResult struct {
	Cursor struct {
		FirstBatch []bson.M `bson:"firstBatch"`
	} `bson:"cursor"`
}

// getIndex returns (index, exists, error).
func getIndex(ctx context.Context, client *mongo.Client, database string, collection string, name string) (mongoIndex, bool, error) {
	result := client.Database(database).RunCommand(ctx, bson.D{{Key: "listIndexes", Value: collection}})
	if err := result.Err(); err != nil {
		// A dropped/nonexistent collection surfaces as NamespaceNotFound (code 26) here
		// rather than an empty list - that's genuine drift (the index, and the
		// collection it lived on, are both gone), not a hard failure. Any other
		// command error (auth, connection, a real server problem) must still be
		// reported, not silently treated as "not found".
		var cmdErr mongo.CommandError
		if errors.As(err, &cmdErr) && cmdErr.Code == 26 {
			return mongoIndex{}, false, nil
		}
		return mongoIndex{}, false, err
	}

	var decoded listIndexesResult
	if err := result.Decode(&decoded); err != nil {
		return mongoIndex{}, false, err
	}

	for _, raw := range decoded.Cursor.FirstBatch {
		if raw["name"] != name {
			continue
		}
		idx := mongoIndex{name: name}
		if keyRaw, ok := raw["key"].(bson.M); ok {
			for field, value := range keyRaw {
				idx.keys = append(idx.keys, bson.E{Key: field, Value: value})
			}
		} else if keyDoc, ok := raw["key"].(bson.D); ok {
			idx.keys = keyDoc
		}
		if hidden, ok := raw["hidden"].(bool); ok {
			idx.hidden = hidden
		}
		if pfe, ok := raw["partialFilterExpression"]; ok {
			b, err := bson.MarshalExtJSON(pfe, true, false)
			if err == nil {
				idx.partialFilterExpression = string(b)
			}
		}
		return idx, true, nil
	}
	return mongoIndex{}, false, nil
}
