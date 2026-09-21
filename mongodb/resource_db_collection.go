package mongodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func resourceDatabaseCollection() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDatabaseCollectionCreate,
		ReadContext:   resourceDatabaseCollectionRead,
		UpdateContext: resourceDatabaseCollectionUpdate,
		DeleteContext: resourceDatabaseCollectionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"db": {
				Type:     schema.TypeString,
				Required: true,
				// A collection can't be moved to a different database in place.
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				// MongoDB has no collection-rename-in-place semantics this provider
				// exposes; a name change is a different collection.
				ForceNew: true,
			},
			"deletion_protection": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Client-side only, never sent to MongoDB. When true (the default), Delete refuses to drop the collection - set it to false first if you actually want `terraform destroy` to remove it.",
			},
			"change_stream_pre_and_post_images": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enables pre- and post-images for change streams on this collection (MongoDB 6.0+). Updatable in place via collMod.",
			},
		},
	}
}

func resourceDatabaseCollectionCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	config := i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	database := data.Get("db").(string)
	name := data.Get("name").(string)
	changeStream := data.Get("change_stream_pre_and_post_images").(bool)

	result := client.Database(database).RunCommand(ctx, bson.D{{Key: "create", Value: name}})
	if result.Err() != nil {
		return diag.Errorf("Could not create the collection : %s ", result.Err())
	}

	if changeStream {
		if err := setCollectionChangeStreamPreAndPostImages(ctx, client, database, name, true); err != nil {
			return diag.Errorf("Could not set change_stream_pre_and_post_images : %s ", err)
		}
	}

	data.SetId(database + "/" + name)
	return resourceDatabaseCollectionRead(ctx, data, i)
}

func resourceDatabaseCollectionRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	config := i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	stateID := data.State().ID
	database, name, err := resourceDatabaseCollectionParseId(stateID)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	changeStream, exists, err := getCollectionChangeStreamPreAndPostImages(ctx, client, database, name)
	if err != nil {
		return diag.Errorf("Error reading collection : %s ", err)
	}
	if !exists {
		// Deleted out of band - treat as drift so Terraform offers to recreate it,
		// rather than hard-erroring forever.
		data.SetId("")
		return nil
	}

	if err := data.Set("db", database); err != nil {
		return diag.Errorf("error setting db : %s ", err)
	}
	if err := data.Set("name", name); err != nil {
		return diag.Errorf("error setting name : %s ", err)
	}
	if err := data.Set("change_stream_pre_and_post_images", changeStream); err != nil {
		return diag.Errorf("error setting change_stream_pre_and_post_images : %s ", err)
	}
	// deletion_protection is client-side only, never stored in MongoDB - leave
	// whatever is already in state/config alone rather than resetting it here.

	data.SetId(stateID)
	return nil
}

func resourceDatabaseCollectionUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	config := i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	database := data.Get("db").(string)
	name := data.Get("name").(string)

	if data.HasChange("change_stream_pre_and_post_images") {
		changeStream := data.Get("change_stream_pre_and_post_images").(bool)
		if err := setCollectionChangeStreamPreAndPostImages(ctx, client, database, name, changeStream); err != nil {
			return diag.Errorf("Could not update change_stream_pre_and_post_images : %s ", err)
		}
	}

	return resourceDatabaseCollectionRead(ctx, data, i)
}

func resourceDatabaseCollectionDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	if data.Get("deletion_protection").(bool) {
		return diag.Errorf("cannot delete collection %q: deletion_protection is true - set it to false first if you really want terraform destroy to drop it", data.Get("name").(string))
	}

	config := i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	database, name, err := resourceDatabaseCollectionParseId(data.State().ID)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	result := client.Database(database).RunCommand(ctx, bson.D{{Key: "drop", Value: name}})
	if result.Err() != nil {
		return diag.Errorf("Could not delete the collection : %s ", result.Err())
	}
	return nil
}

func resourceDatabaseCollectionParseId(id string) (string, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected database/name", id)
	}
	return parts[0], parts[1], nil
}

func setCollectionChangeStreamPreAndPostImages(ctx context.Context, client *mongo.Client, database string, name string, enabled bool) error {
	result := client.Database(database).RunCommand(ctx, bson.D{
		{Key: "collMod", Value: name},
		{Key: "changeStreamPreAndPostImages", Value: bson.D{{Key: "enabled", Value: enabled}}},
	})
	return result.Err()
}

type listCollectionsResult struct {
	Cursor struct {
		FirstBatch []struct {
			Name    string `bson:"name"`
			Options struct {
				ChangeStreamPreAndPostImages struct {
					Enabled bool `bson:"enabled"`
				} `bson:"changeStreamPreAndPostImages"`
			} `bson:"options"`
		} `bson:"firstBatch"`
	} `bson:"cursor"`
}

// getCollectionChangeStreamPreAndPostImages returns (enabled, exists, error).
func getCollectionChangeStreamPreAndPostImages(ctx context.Context, client *mongo.Client, database string, name string) (bool, bool, error) {
	result := client.Database(database).RunCommand(ctx, bson.D{
		{Key: "listCollections", Value: 1},
		{Key: "filter", Value: bson.D{{Key: "name", Value: name}}},
	})
	var decoded listCollectionsResult
	if err := result.Decode(&decoded); err != nil {
		return false, false, err
	}
	if len(decoded.Cursor.FirstBatch) == 0 {
		return false, false, nil
	}
	return decoded.Cursor.FirstBatch[0].Options.ChangeStreamPreAndPostImages.Enabled, true, nil
}
