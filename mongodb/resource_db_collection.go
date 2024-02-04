package mongodb

import (
	"context"
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
				ForceNew: true,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"deletion_protection": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceDatabaseCollectionDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var stateId = data.State().ID
	var db = data.Get("db").(string)

	// StateID is a concatenation of database and collection name. We only use the collection here.
	_, collectionName, err :=resourceDatabaseCollectionParseId(stateId)
	if err != nil {
		return diag.Errorf("ID mismatch %s", err)
	}

	_err := dropCollection(client, db, collectionName, data)
	if _err != nil {
		return _err
	}

	return nil
}

func resourceDatabaseCollectionUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}

	var stateId = data.State().ID
	_ , errEncoding := ParseId(stateId, 2)
	if errEncoding != nil {
		return diag.Errorf("ID mismatch %s", errEncoding)
	}

	var collectionName = data.Get("name").(string)
	var db = data.Get("db").(string)

	dbClient := client.Database(db)
	_err := dbClient.CreateCollection(context.Background(), collectionName + "bla")
	if _err != nil {
		return diag.Errorf("%s", _err)
	}

	SetId(data, []string{db, collectionName})
	return resourceDatabaseCollectionRead(ctx, data, i)
}

func resourceDatabaseCollectionRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	stateID := data.State().ID
	db, collectionName, err := resourceDatabaseCollectionParseId(stateID)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	dbClient := client.Database(db)

	// Construct the filter to check if collection exists
	filter := bson.M{"name": collectionName}

	// List the collections with the specified name
	cursor, err := dbClient.ListCollections(context.Background(), filter)
	if err != nil {
		return diag.Errorf("Failed to list collections : %s ", err)
	}

	// Check if the collection exists
	exists := cursor.Next(context.Background())
	if !exists {
		return diag.Errorf("collection does not exist")
	}

	dataSetError := data.Set("db", db)
	if dataSetError != nil {
		return diag.Errorf("error setting database : %s ", dataSetError)
	}
	dataSetError = data.Set("name", collectionName)
	if dataSetError != nil {
		return diag.Errorf("error setting collection name : %s ", dataSetError)
	}
	data.SetId(stateID)
	return nil
}

func resourceDatabaseCollectionCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to db : %s ", connectionError)
	}
	var db = data.Get("db").(string)
	var collectionName = data.Get("name").(string)

	dbClient := client.Database(db)

	err := dbClient.CreateCollection(context.Background(), collectionName)
	if err != nil {
		return diag.Errorf("Could not create the collection : %s ", err)
	}

	SetId(data, []string{db, collectionName})
	return resourceDatabaseCollectionRead(ctx, data, i)
}

func dropCollection(client *mongo.Client, db string, collectionName string, data *schema.ResourceData) diag.Diagnostics {
	var deletionProtection = data.Get("deletion_protection").(bool)
	if deletionProtection == true {
		return diag.Errorf("Can't delete collection because deletion protection is enabled")
	}

	dbClient := client.Database(db)
	collectionClient := dbClient.Collection(collectionName)
	err := collectionClient.Drop(context.Background())
	if err != nil {
		return diag.Errorf("%s", err)
	}

	return nil
}

func resourceDatabaseCollectionParseId(id string) (string, string, error) {
	parts, err := ParseId(id, 2)
	if err != nil {
		return "", "", err
	}

	db := parts[0]
	collectionName := parts[1]
	return db, collectionName, nil
}
