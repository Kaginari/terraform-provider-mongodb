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
				Default:  "admin",
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

func readRoleFromData(data *schema.ResourceData) (*Role, error) {
	var roleName = data.Get("name").(string)
	var database = data.Get("database").(string)
	var roleList []RoleReference
	var privileges []Privilege

	privilege := data.Get("privilege").(*schema.Set).List()
	roles := data.Get("inherited_role").(*schema.Set).List()

	roleMapErr := mapstructure.Decode(roles, &roleList)
	if roleMapErr != nil {
		return nil, roleMapErr
	}
	privMapErr := mapstructure.Decode(privilege, &privileges)
	if privMapErr != nil {
		return nil, privMapErr
	}

	role := Role{
		Name:       roleName,
		Database:   database,
		Roles:      roleList,
		Privileges: privileges,
	}

	return &role, nil
}

func writeRoleToData(data *schema.ResourceData, role *Role) error {
	if role == nil {
		data.SetId("")
		return nil
	}

	inheritedRoles := make([]interface{}, len(role.Roles))
	for i, s := range role.Roles {
		inheritedRoles[i] = map[string]interface{}{
			"db":   s.Db,
			"role": s.Role,
		}
	}

	privileges := make([]interface{}, len(role.Privileges))
	for i, s := range role.Privileges {
		privileges[i] = map[string]interface{}{
			"db":         s.Db,
			"collection": s.Collection,
			"actions":    s.Actions,
		}
	}
	err := data.Set("inherited_role", inheritedRoles)
	if err != nil {
		return err
	}
	err = data.Set("privilege", privileges)
	if err != nil {
		return err
	}
	err = data.Set("database", role.Database)
	if err != nil {
		return err
	}
	err = data.Set("name", role.Name)
	if err != nil {
		return err
	}
	data.SetId(makeRoleId(role.Name, role.Database))
	return nil
}

func resourceDatabaseRoleCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}

	role, readRoleErr := readRoleFromData(data)
	if readRoleErr != nil {
		return diag.Errorf("Error reading role : %s ", readRoleErr)
	}

	createRoleErr := createRole(client, role)
	if createRoleErr != nil {
		return diag.Errorf("Could not create the role : %s ", createRoleErr)
	}

	writeRoleErr := writeRoleToData(data, role)
	if writeRoleErr != nil {
		return diag.Errorf("Error writing role : %s ", writeRoleErr)
	}

	return nil
}

func resourceDatabaseRoleDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}

	roleName, database, parseRoleIdErr := parseRoleId(data.State().ID)
	if parseRoleIdErr != nil {
		return diag.Errorf("%s", parseRoleIdErr)
	}

	dropRoleErr := dropRole(client, roleName, database)
	if dropRoleErr != nil {
		return diag.Errorf("Error deleting the role: %s ", dropRoleErr)
	}

	return nil
}

func resourceDatabaseRoleUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}

	_, _, parseRoleIdErr := parseRoleId(data.State().ID)
	if parseRoleIdErr != nil {
		return diag.Errorf("%s", parseRoleIdErr)
	}

	role, readRoleErr := readRoleFromData(data)
	if readRoleErr != nil {
		return diag.Errorf("Error reading role : %s ", readRoleErr)
	}

	dropRoleErr := dropRole(client, role.Name, role.Database)
	if dropRoleErr != nil {
		return diag.Errorf("Error deleting the role: %s ", dropRoleErr)
	}

	createRoleErr := createRole(client, role)
	if createRoleErr != nil {
		return diag.Errorf("Could not create the role  :  %s ", createRoleErr)
	}

	writeRoleErr := writeRoleToData(data, role)
	if writeRoleErr != nil {
		return diag.Errorf("Error writing role : %s ", writeRoleErr)
	}

	return nil
}

func resourceDatabaseRoleRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client, connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}

	roleName, database, parseRoleIdErr := parseRoleId(data.State().ID)
	if parseRoleIdErr != nil {
		return diag.Errorf("%s", parseRoleIdErr)
	}

	role, decodeError := getRole(client, roleName, database)
	if decodeError != nil {
		return diag.Errorf("Error decoding role : %s ", decodeError)
	}

	writeRoleErr := writeRoleToData(data, role)
	if writeRoleErr != nil {
		return diag.Errorf("Error writing role : %s ", writeRoleErr)
	}

	return nil
}

func parseRoleId(id string) (string, string, error) {
	result, errEncoding := base64.StdEncoding.DecodeString(id)

	if errEncoding != nil {
		return "", "", fmt.Errorf("unexpected format of ID Error : %s", errEncoding)
	}
	parts := strings.SplitN(string(result), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected database.roleName", id)
	}

	database := parts[0]
	roleName := parts[1]

	return roleName, database, nil
}

func makeRoleId(role string, database string) string {
	str := database + "." + role
	return base64.StdEncoding.EncodeToString([]byte(str))
}
