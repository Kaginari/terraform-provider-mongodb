package mongodb

import (
	"context"
	"encoding/hex"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"time"
	"strings"
)

func resourceDatabaseUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDatabaseUserCreate,
		ReadContext:   resourceDatabaseUserRead,
		UpdateContext: resourceDatabaseUserUpdate,
		DeleteContext: resourceDatabaseUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importDatabaseUserState,
		},
		Schema: map[string]*schema.Schema{
			"auth_database": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name":{
				Type:     schema.TypeString,
				Required: true,
			},
			"password":{
				Type:     schema.TypeString,
				Required: true,
			},
			"role": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 10,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"db": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"role": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}



func resourceDatabaseUserDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var client = i.(*mongo.Client)
	var stateId = data.State().ID
	var database = data.Get("auth_database").(string)

	id, errEncoding := hex.DecodeString(stateId)
	if errEncoding != nil {
		return diag.Errorf("ID mismatch %s", errEncoding)
	}

	// StateID is a concatination of database and username. We only use the username here.
	splitId := strings.Split(string(id), ".")
	userName := splitId[1]

	adminDB := client.Database(database)

	result := adminDB.RunCommand(context.Background(), bson.D{{Key: "dropUser", Value: userName}})
	if result.Err() != nil {
		return diag.Errorf("%s",result.Err())
	}

	return resourceDatabaseUserRead(ctx, data, i)
}

func resourceDatabaseUserUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var client = i.(*mongo.Client)

	var stateId = data.State().ID
	_, errEncoding := hex.DecodeString(stateId)
	if errEncoding != nil {
		return diag.Errorf("ID mismatch %s", errEncoding)
	}

	var userName = data.Get("name").(string)
	var database = data.Get("auth_database").(string)
	var userPassword = data.Get("password").(string)
	
	adminDB := client.Database(database)

	result := adminDB.RunCommand(context.Background(), bson.D{{Key: "dropUser", Value: userName}})
	if result.Err() != nil {
		return diag.Errorf("%s",result.Err())
	}
	var roleList []Role
	var user = DbUser{
		Name:     userName,
		Password: userPassword,
	}
	roles := data.Get("role").(*schema.Set).List()
	roleMapErr := mapstructure.Decode(roles, &roleList)
	if roleMapErr != nil {
		return diag.Errorf("Error decoding map : %s ", roleMapErr)
	}
	err2 := createUser(client,user,roleList,database)
	if err2 != nil {
		return diag.Errorf("Could not create the user : %s ", err2)
	}

	newId := database+"."+userName
	hexadecimal := hex.EncodeToString([]byte(newId))
	data.SetId(hexadecimal)
	return resourceDatabaseUserRead(ctx, data, i)
}

func resourceDatabaseUserRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	diags = nil
	return diags
}

func resourceDatabaseUserCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {

	var client = i.(*mongo.Client)
	var database = data.Get("auth_database").(string)
	var userName = data.Get("name").(string)
	var userPassword = data.Get("password").(string)
	var roleList []Role
	var user = DbUser{
		Name:     userName,
		Password: userPassword,
	}
	roles := data.Get("role").(*schema.Set).List()
	roleMapErr := mapstructure.Decode(roles, &roleList)
	if roleMapErr != nil {
		return diag.Errorf("Error decoding map : %s ", roleMapErr)
	}
	err := createUser(client,user,roleList,database)
	if err != nil {
		return diag.Errorf("Could not create the user : %s ", err)
	}
	str := database+"."+userName
	hx := hex.EncodeToString([]byte(str))
	data.SetId(hx)
	return resourceDatabaseUserRead(ctx, data, i)
}

func importDatabaseUserState(ctx context.Context, data *schema.ResourceData, i interface{}) ([]*schema.ResourceData, error) {
	if err := data.Set("auth_database", data.Get("auth_database")); err != nil {
		return nil, err
	}
	data.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return []*schema.ResourceData{data}, nil
}
