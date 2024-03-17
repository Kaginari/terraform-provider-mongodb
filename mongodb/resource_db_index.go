package mongodb

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
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
				ForceNew: true,
				Required: true,
			},
			"collection": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"keys": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
					},
				},
			},
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if len(old) > 0 && len(new) == 0 {
						return true
					}
					return false
				},
			},
			//"unique": {
			//	Type:     schema.TypeBool,
			//	Optional: true,
			//	Default:  false,
			//},
			//"sparse": {
			//	Type:     schema.TypeBool,
			//	Optional: true,
			//	Default:  false,
			//},
			//"bits": {
			//	Type:     schema.TypeInt,
			//	Optional: true,
			//	Default:  26,
			//},
			//"max": {
			//	Type:     schema.TypeFloat,
			//	Optional: true,
			//	Default:  180.0,
			//},
			//"min": {
			//	Type:     schema.TypeFloat,
			//	Optional: true,
			//	Default:  -180.0,
			//},
			"timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  30,
			},
		},
	}
}

func resourceDatabaseIndexCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to db : %s ", connectionError)
	}
	var db = data.Get("db").(string)
	var collectionName = data.Get("collection").(string)

	indexName, err := createIndex(client, db, collectionName, data)
	if err != nil {
		return err
	}

	SetId(data, []string{db, collectionName, indexName})
	return resourceDatabaseIndexRead(ctx, data, i)
}

func resourceDatabaseIndexRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	stateID := data.State().ID

	db, collectionName, indexName, err := resourceDatabaseIndexParseId(stateID)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	collectionClient := client.Database(db).Collection(collectionName)
	if collectionClient == nil {
		return diag.Errorf("Collection client is nil")
	}

	// Get all indexes for the collection
	indexes, err := collectionClient.Indexes().List(context.Background())
	if err != nil {
		return diag.Errorf("Failed to list indexes: %s", err)
	}

	var results []bson.M
	if err = indexes.All(context.Background(), &results); err != nil {
		{
			return diag.Errorf("Failed to list indexes: %s", err)
		}
	}

	indexFound := false
	keys := make(map[string]string)
	for _, result := range results {
		tflog.Debug(ctx, fmt.Sprintf("Index: %v", result))
		for k, v := range result {
			if k == "name" && v == indexName {
				keysPrimitives := result["key"].(primitive.M)
				for key, value := range keysPrimitives {
					keys[key] = fmt.Sprintf("%v", value)
				}
				indexFound = true
				break
			}
			if indexFound {
				break
			}
		}
	}

	if !indexFound {
		return diag.Errorf("index does not exist")
	}

	_ = data.Set("db", db)
	_ = data.Set("collection", collectionName)
	_ = data.Set("name", indexName)
	_ = data.Set("keys", keys)
	_ = data.Set("timeout", data.Get("timeout").(int))

	return nil
}

func resourceDatabaseIndexUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	return resourceDatabaseIndexRead(ctx, data, i)
}

func resourceDatabaseIndexDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}

	// StateID is a concatenation of database and collection name. We only use the collection & index here.
	db, collectionName, indexName, err := resourceDatabaseIndexParseId(data.State().ID)
	if err != nil {
		return diag.Errorf("Failed to parse index ID %s", err)
	}

	_err := dropIndex(client, db, collectionName, indexName)
	if _err != nil {
		return _err
	}

	return nil
}

func createIndex(client *mongo.Client, db string, collectionName string, data *schema.ResourceData) (string, diag.Diagnostics) {
	collectionClient := client.Database(db).Collection(collectionName)

	var keys = data.Get("keys").([]interface{})

	// Create the index keys
	indexKeys := bson.D{}
	for _, _key := range keys {
		key := _key.(map[string]interface{})
		keyField := key["field"].(string)
		value := key["value"].(string)
		if value == "1" {
			indexKeys = append(indexKeys, bson.E{Key: keyField, Value: 1})
		} else if value == "-1" {
			indexKeys = append(indexKeys, bson.E{Key: keyField, Value: -1})
		} else if value == "true" {
			indexKeys = append(indexKeys, bson.E{Key: keyField, Value: true})
		} else if value == "false" {
			indexKeys = append(indexKeys, bson.E{Key: keyField, Value: false})
		} else {
			indexKeys = append(indexKeys, bson.E{Key: keyField, Value: value})
		}
	}

	// Initialize options.Index
	indexOptions := options.Index()
	//indexOptions.SetUnique(data.Get("unique").(bool))
	//indexOptions.SetSparse(data.Get("sparse").(bool))
	//indexOptions.SetBits(int32(data.Get("bits").(int)))
	//indexOptions.SetMin(data.Get("min").(float64))
	//indexOptions.SetMax(data.Get("max").(float64))
	var name = data.Get("name").(string)
	if len(name) > 0 {
		indexOptions.SetName(name)
	}

	// Create the index model
	indexModel := mongo.IndexModel{
		Keys:    indexKeys,
		Options: indexOptions,
	}

	var timeout = data.Get("timeout").(int)
	opts := options.CreateIndexes().SetMaxTime(time.Duration(timeout) * time.Second)

	// Create the index
	indexName, err := collectionClient.Indexes().CreateOne(context.Background(), indexModel, opts)
	if err != nil {
		return "", diag.Errorf("Could not create the index : %s ", err)
	}
	return indexName, nil
}

func dropIndex(client *mongo.Client, db string, collectionName string, indexName string) diag.Diagnostics {
	dbClient := client.Database(db)
	collectionClient := dbClient.Collection(collectionName)
	_, err := collectionClient.Indexes().DropOne(context.TODO(), indexName)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	return nil
}

func resourceDatabaseIndexParseId(id string) (string, string, string, error) {
	parts, err := ParseId(id, 3)
	if err != nil {
		return "", "", "", err
	}

	db := parts[0]
	collectionName := parts[1]
	indexName := parts[2]
	return db, collectionName, indexName, nil
}
