package mongodb

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
	"go.mongodb.org/mongo-driver/bson"
	"strings"
)

func resourceDatabaseRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDatabaseRoleCreate,
		ReadContext:   resourceDatabaseRoleRead,
		UpdateContext: resourceDatabaseRoleUpdate,
		DeleteContext: resourceDatabaseRoleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"database": {
				Type:     schema.TypeString,
				Optional: true,
				Default: "admin",
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"privilege": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 10,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"db": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"collection": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"actions": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"inherited_role": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 2,
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

func resourceDatabaseRoleCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client , connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var role = data.Get("name").(string)
	var database = data.Get("database").(string)
	var roleList []Role
	var privileges []PrivilegeDto

	privilege := data.Get("privilege").(*schema.Set).List()
	roles := data.Get("inherited_role").(*schema.Set).List()

	roleMapErr := mapstructure.Decode(roles, &roleList)
	if roleMapErr != nil {
		return diag.Errorf("Error decoding map : %s ", roleMapErr)
	}
	privMapErr := mapstructure.Decode(privilege, &privileges)
	if privMapErr != nil {
		return diag.Errorf("Error decoding map : %s ", privMapErr)
	}


	err := createRole(client, role, roleList, privileges, database)

	if err != nil {
		return diag.Errorf("Could not create the role : %s ", err)
	}
	str := database+"."+role
	encoded := base64.StdEncoding.EncodeToString([]byte(str))
	data.SetId(encoded)
	return resourceDatabaseRoleRead(ctx, data, i)
}

func resourceDatabaseRoleDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client , connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var stateId = data.State().ID
	roleName, database , err := resourceDatabaseRoleParseId(stateId)

	if err != nil {
		return diag.Errorf("%s", err)
	}

	db := client.Database(database)
	result := db.RunCommand(context.Background(), bson.D{{Key: "dropRole", Value: roleName}})

	if result.Err() != nil {
		return diag.Errorf("%s",result.Err())
	}

	return nil
}

func resourceDatabaseRoleUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client , connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var role = data.Get("name").(string)
	var stateId = data.State().ID
	roleName, database , err := resourceDatabaseRoleParseId(stateId)

	if err != nil {
		return diag.Errorf("%s",err)
	}

	db := client.Database(database)
	result := db.RunCommand(context.Background(), bson.D{{Key: "dropRole", Value: roleName}})

	if result.Err() != nil {
		return diag.Errorf("%s", result.Err())
	}

	var roleList []Role
	var privileges []PrivilegeDto

	privilege := data.Get("privilege").(*schema.Set).List()
	roles := data.Get("inherited_role").(*schema.Set).List()

	roleMapErr := mapstructure.Decode(roles, &roleList)
	if roleMapErr != nil {
		return diag.Errorf("Error decoding map : %s ", roleMapErr)
	}
	privMapErr := mapstructure.Decode(privilege, &privileges)
	if privMapErr != nil {
		return diag.Errorf("Error decoding map : %s ", privMapErr)
	}

	err2 := createRole(client, role, roleList, privileges, database)

	if err2 != nil {
		return diag.Errorf("Could not create the role  :  %s ", err)
	}
	str := database+"."+role
	encoded := base64.StdEncoding.EncodeToString([]byte(str))
	data.SetId(encoded)


	return resourceDatabaseRoleRead(ctx, data, i)
}

func resourceDatabaseRoleRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	var config = i.(*MongoDatabaseConfiguration)
	client , connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	stateID := data.State().ID
	roleName, database , err := resourceDatabaseRoleParseId(stateID)
	if err != nil {
		return diag.Errorf("%s",err)
	}
	result , decodeError := getRole(client,roleName,database)
	if decodeError != nil {
		return diag.Errorf("Error decoding role : %s ", err)
	}
	if len(result.Roles) == 0 {
		return diag.Errorf("Role does not exist")
	}
	inheritedRoles := make([]interface{}, len(result.Roles[0].InheritedRoles))

	for i, s := range result.Roles[0].InheritedRoles {
		inheritedRoles[i] = map[string]interface{}{
			"db": s.Db,
			"role": s.Role,
		}
	}
	dataSetError := data.Set("inherited_role", inheritedRoles)
	if dataSetError != nil {
		return diag.Errorf("Error setting  inherited roles : %s ", err)
	}
	privileges := make([]interface{}, len(result.Roles[0].Privileges))

	for i, s := range result.Roles[0].Privileges {
		privileges[i] = map[string]interface{}{
			"db": s.Resource.Db,
			"collection": s.Resource.Collection,
			"actions": s.Actions,
		}
	}
	dataSetError = data.Set("privilege", privileges)
	if dataSetError != nil {
		return diag.Errorf("Error setting role privilege : %s ", err)
	}
	dataSetError = data.Set("database", database)
	if dataSetError != nil {
		return diag.Errorf("Error setting role database : %s ", err)
	}
	dataSetError = data.Set("name", roleName)
	if dataSetError != nil {
		return diag.Errorf("Error setting  role nam: %s ", err)
	}

	data.SetId(stateID)
	diags = nil
	return diags
}

func resourceDatabaseRoleParseId(id string) (string, string, error) {
	result , errEncoding := base64.StdEncoding.DecodeString(id)

	if errEncoding != nil {
		return "", "", fmt.Errorf("unexpected format of ID Error : %s", errEncoding)
	}
	parts := strings.SplitN(string(result), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected database.roleName", id)
	}

	database := parts[0]
	roleName := parts[1]

	return roleName , database , nil
}

