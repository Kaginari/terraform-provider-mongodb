package mongodb

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
)

func resourceDatabaseUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDatabaseUserCreate,
		ReadContext:   resourceDatabaseUserRead,
		UpdateContext: resourceDatabaseUserUpdate,
		DeleteContext: resourceDatabaseUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"auth_database": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"password": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 25,
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
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var database = data.Get("auth_database").(string)

	userName, database, parseUserIdErr := parseUserId(data.State().ID)
	if parseUserIdErr != nil {
		return diag.Errorf("ID mismatch %s", parseUserIdErr)
	}

	deleteUserErr := deleteUser(client, userName, database)
	if deleteUserErr != nil {
		return diag.Errorf("Could not delete the user : %s ", deleteUserErr)
	}

	return nil
}

func resourceDatabaseUserUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	_, _, parseUserIdErr := parseUserId(data.State().ID)
	if parseUserIdErr != nil {
		return diag.Errorf("ID mismatch %s", parseUserIdErr)
	}

	var userName = data.Get("name").(string)
	var database = data.Get("auth_database").(string)
	var userPassword = data.Get("password").(string)

	var roleList []Role
	roles := data.Get("role").(*schema.Set).List()
	roleMapErr := mapstructure.Decode(roles, &roleList)
	if roleMapErr != nil {
		return diag.Errorf("Error decoding map : %s ", roleMapErr)
	}

	var user = DbUser{
		Name:     userName,
		Password: userPassword,
	}

	deleteUserErr := deleteUser(client, userName, database)
	if deleteUserErr != nil {
		return diag.Errorf("Could not delete the user : %s ", deleteUserErr)
	}

	createUserErr := createUser(client, user, roleList, database)
	if createUserErr != nil {
		return diag.Errorf("Could not create the user : %s ", createUserErr)
	}

	data.SetId(makeUserId(userName, database))
	return resourceDatabaseUserRead(ctx, data, i)
}

func resourceDatabaseUserRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	username, database, parseUserIdErr := parseUserId(data.State().ID)
	password := data.Get("password")
	if parseUserIdErr != nil {
		return diag.Errorf("Error parsing user id : %s ", parseUserIdErr)
	}
	result, decodeError := getUser(client, username, database)
	if decodeError != nil {
		return diag.Errorf("Error decoding user : %s ", decodeError)
	}
	if len(result.Users) == 0 {
		return diag.Errorf("User %s.%s does not exist", database, username)
	}
	roles := make([]interface{}, len(result.Users[0].Roles))

	for i, s := range result.Users[0].Roles {
		roles[i] = map[string]interface{}{
			"db":   s.Db,
			"role": s.Role,
		}
	}

	dataSetError := data.Set("role", roles)
	if dataSetError != nil {
		return diag.Errorf("error setting role : %s ", dataSetError)
	}
	dataSetError = data.Set("auth_database", database)
	if dataSetError != nil {
		return diag.Errorf("error setting auth_db : %s ", dataSetError)
	}
	dataSetError = data.Set("password", password)
	if dataSetError != nil {
		return diag.Errorf("error setting password : %s ", dataSetError)
	}
	data.SetId(makeUserId(username, database))
	return nil
}

func resourceDatabaseUserCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}

	var database = data.Get("auth_database").(string)
	var userName = data.Get("name").(string)
	var userPassword = data.Get("password").(string)
	var roleList []Role
	roles := data.Get("role").(*schema.Set).List()
	roleMapErr := mapstructure.Decode(roles, &roleList)
	if roleMapErr != nil {
		return diag.Errorf("Error decoding map : %s ", roleMapErr)
	}

	var user = DbUser{
		Name:     userName,
		Password: userPassword,
	}
	err := createUser(client, user, roleList, database)
	if err != nil {
		return diag.Errorf("Could not create the user : %s ", err)
	}
	data.SetId(makeUserId(userName, database))
	return resourceDatabaseUserRead(ctx, data, i)
}

func parseUserId(id string) (string, string, error) {
	result, errEncoding := base64.StdEncoding.DecodeString(id)

	if errEncoding != nil {
		return "", "", fmt.Errorf("unexpected format of ID Error : %s", errEncoding)
	}
	parts := strings.SplitN(string(result), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected db.username", id)
	}

	database := parts[0]
	userName := parts[1]

	return userName, database, nil
}

func makeUserId(userName string, database string) string {
	str := database + "." + userName
	return base64.StdEncoding.EncodeToString([]byte(str))
}
