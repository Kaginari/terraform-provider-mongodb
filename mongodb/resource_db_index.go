package mongodb

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"strings"
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
				Type: schema.TypeMap,
				Required: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"unique": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"sparse": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"bits": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  26,
			},
			"max": {
				Type:     schema.TypeFloat,
				Optional: true,
				Default:  180.0,
			},
			"min": {
				Type:     schema.TypeFloat,
				Optional: true,
				Default:  -180.0,
			},
			"timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  30,
			},
		},
	}
}

func resourceDatabaseIndexDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var stateId = data.State().ID
	var db = data.Get("db").(string)

	// StateID is a concatenation of database and collection name. We only use the collection & index here.
	_, collectionName, indexName, err := resourceDatabaseIndexParseId(stateId)
	if err != nil {
		return diag.Errorf("Failed to parse index ID %s", err)
	}

	_err := dropIndex(client, db, collectionName, indexName)
	if _err != nil {
		return _err
	}

	return nil
}

func resourceDatabaseIndexUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var stateId = data.State().ID
	_, errEncoding := base64.StdEncoding.DecodeString(stateId)
	if errEncoding != nil {
		return diag.Errorf("ID mismatch %s", errEncoding)
	}

	var indexName = data.Get("name").(string)
	var collectionName = data.Get("collection").(string)
	var db = data.Get("db").(string)

	err := dropIndex(client, db, collectionName, indexName)
	if err != nil {
		return err
	}

	indexName, err = createIndex(client, db, collectionName, data)
	if err != nil {
		return err
	}

	setId(data, db, collectionName, indexName)
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
		log.Fatal(err)
	}

	indexFound := false
	for _, result := range results {
		for k, v := range result {
			if k == "name" && v == indexName {
				indexFound = true
				break
			}
		}
	}

	if !indexFound {
		return diag.Errorf("index does not exist")
	}

	//if len(results) > 0 {
	//	return diag.Errorf("INDEX NAME: %+v\n", results)
	//}

	// Find the index with the specified name
	//var index *mongo.IndexModel
	//for indexes.Next(context.Background()) {
	//	var currentIndex mongo.IndexModel
	//	err := indexes.Decode(&currentIndex)
	//	if err != nil {
	//		return diag.Errorf("Failed to decode index model: %s", err)
	//	}
	//
	//	if currentIndex.Options != nil && currentIndex.Options.Name != nil && *currentIndex.Options.Name == indexName {
	//		index = &currentIndex
	//		break
	//	}
	//}
	//
	//if index == nil {
	//	return diag.Errorf("index does not exist")
	//}

	dataSetError := data.Set("db", db)
	if dataSetError != nil {
		return diag.Errorf("error setting database : %s ", dataSetError)
	}
	dataSetError = data.Set("collection", collectionName)
	if dataSetError != nil {
		return diag.Errorf("error setting collection name : %s ", dataSetError)
	}
	dataSetError = data.Set("name", indexName)
	if dataSetError != nil {
		return diag.Errorf("error setting index name : %s ", dataSetError)
	}
	data.SetId(stateID)
	return nil
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

	setId(data, db, collectionName, indexName)
	return resourceDatabaseIndexRead(ctx, data, i)
}

func createIndex(client *mongo.Client, db string, collectionName string, data *schema.ResourceData) (string, diag.Diagnostics) {
	collectionClient := client.Database(db).Collection(collectionName)

	var keys = data.Get("keys").(map[string]interface{})

	// Create the index keys
	indexKeys := bson.D{}
	for key, value := range keys {
		valueStr := value.(string)
		if valueStr == "1" {
			indexKeys = append(indexKeys, bson.E{Key: key, Value: 1})
		} else if valueStr == "-1" {
			indexKeys = append(indexKeys, bson.E{Key: key, Value: -1})
		} else {
			indexKeys = append(indexKeys, bson.E{Key: key, Value: valueStr})
		}
	}

	// Initialize options.Index
	indexOptions := options.Index()
	indexOptions.SetUnique(data.Get("unique").(bool))
	indexOptions.SetSparse(data.Get("sparse").(bool))
	indexOptions.SetBits(int32(data.Get("bits").(int)))
	indexOptions.SetMin(data.Get("min").(float64))
	indexOptions.SetMax(data.Get("max").(float64))
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

func setId(data *schema.ResourceData, db string, collectionName string, indexName string) {
	id := db + "." + collectionName + "." + indexName
	encoded := base64.StdEncoding.EncodeToString([]byte(id))
	data.SetId(encoded)
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
	result, errEncoding := base64.StdEncoding.DecodeString(id)

	if errEncoding != nil {
		return "", "", "", fmt.Errorf("unexpected format of ID Error : %s", errEncoding)
	}
	parts := strings.SplitN(string(result), ".", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("unexpected format of ID (%s), expected attribute1.attribute2.attribute3", id)
	}

	db := parts[0]
	collectionName := parts[1]
	indexName := parts[2]
	return db, collectionName, indexName, nil
}
