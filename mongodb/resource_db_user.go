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
				Optional: true,
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

func readUserFromData(data *schema.ResourceData) (*User, error) {
	userName := data.Get("name").(string)
	database := data.Get("auth_database").(string)
	password := data.Get("password").(string)
	rolesData := data.Get("role").(*schema.Set).List()

	var roles []RoleReference
	roleMapErr := mapstructure.Decode(rolesData, &roles)
	if roleMapErr != nil {
		return nil, roleMapErr
	}

	if password == "" && database != "$external" {
		return nil, fmt.Errorf("users without password allowed only for X509 certificate users that have to be in the $external database, but database %s was specified", database)
	}

	var user = User{
		AuthDatabase: database,
		Name:         userName,
		Password:     password,
		Roles:        roles,
	}

	return &user, nil
}

func writeUserToData(data *schema.ResourceData, user *User) error {
	if user == nil {
		data.SetId("")
		return nil
	}

	roles := make([]interface{}, len(user.Roles))
	for i, s := range user.Roles {
		roles[i] = map[string]interface{}{"db": s.Db, "role": s.Role}
	}

	err := data.Set("role", roles)
	if err != nil {
		return err
	}
	err = data.Set("auth_database", user.AuthDatabase)
	if err != nil {
		return err
	}
	err = data.Set("password", user.Password)
	if err != nil {
		return err
	}

	data.SetId(makeUserId(user.Name, user.AuthDatabase))
	return nil
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

	deleteUserErr := dropUser(client, userName, database)
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
	userName, database, parseUserIdErr := parseUserId(data.State().ID)
	if parseUserIdErr != nil {
		return diag.Errorf("ID mismatch %s", parseUserIdErr)
	}

	user, convertUserErr := readUserFromData(data)
	if convertUserErr != nil {
		return diag.Errorf("Error reading user : %s ", convertUserErr)
	}

	deleteUserErr := dropUser(client, userName, database)
	if deleteUserErr != nil {
		return diag.Errorf("Could not delete the user : %s ", deleteUserErr)
	}

	createUserErr := createUser(client, user)
	if createUserErr != nil {
		return diag.Errorf("Could not create the user : %s ", createUserErr)
	}

	writeUserErr := writeUserToData(data, user)
	if writeUserErr != nil {
		return diag.Errorf("Error writing user : %s ", writeUserErr)
	}

	return nil
}

func resourceDatabaseUserRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}

	userName, database, parseUserIdErr := parseUserId(data.State().ID)
	password := data.Get("password").(string)
	if parseUserIdErr != nil {
		return diag.Errorf("Error parsing user id : %s ", parseUserIdErr)
	}

	user, decodeError := getUser(client, userName, database, password)
	if decodeError != nil {
		return diag.Errorf("Error decoding user : %s ", decodeError)
	}

	writeUserErr := writeUserToData(data, user)
	if writeUserErr != nil {
		return diag.Errorf("Error writing user : %s ", writeUserErr)
	}

	return nil
}

func resourceDatabaseUserCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}

	user, convertUserErr := readUserFromData(data)
	if convertUserErr != nil {
		return diag.Errorf("Error reading user : %s ", convertUserErr)
	}

	createUserErr := createUser(client, user)
	if createUserErr != nil {
		return diag.Errorf("Could not create the user : %s ", createUserErr)
	}

	writeUserErr := writeUserToData(data, user)
	if writeUserErr != nil {
		return diag.Errorf("Error writing user : %s ", writeUserErr)
	}

	return nil
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
