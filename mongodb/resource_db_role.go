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
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceDatabaseRoleResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceDatabaseRoleUpgradeV0,
			},
		},
		Schema: map[string]*schema.Schema{
			"database": {
				Type:     schema.TypeString,
				Optional: true,
				Default: "admin",
				// MongoDB's updateRole command cannot move a role to a different
				// database - that's a different role identity. Force a create/destroy
				// instead of silently targeting the wrong role on update.
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				// MongoDB has no role-rename command; a name change is a new identity.
				ForceNew: true,
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

// resourceDatabaseRoleResourceV0 is the schema shape prior to the "database/name" ID
// migration (v0 IDs were base64("database.name")). It only needs to describe the schema
// closely enough for StateUpgraders to decode the raw v0 state, so it mirrors the current
// schema rather than being maintained as a historical snapshot.
func resourceDatabaseRoleResourceV0() *schema.Resource {
	return &schema.Resource{
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

// resourceDatabaseRoleUpgradeV0 rewrites the base64("database.name") ID from schema v0 into
// the plain "database/name" ID used from v1 onward, so the provider upgrade is invisible:
// existing state is migrated in place on the next refresh/apply, with no forced replacement
// and no manual `terraform state` surgery required from the user.
func resourceDatabaseRoleUpgradeV0(_ context.Context, rawState map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
	oldId, ok := rawState["id"].(string)
	if !ok || oldId == "" {
		return rawState, nil
	}

	decoded, errEncoding := base64.StdEncoding.DecodeString(oldId)
	if errEncoding != nil {
		// Already in the new plain format (or something else migrated it already); leave as-is.
		return rawState, nil
	}

	parts := strings.SplitN(string(decoded), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return rawState, fmt.Errorf("unexpected format of v0 ID (%s), expected base64(database.roleName)", oldId)
	}

	rawState["id"] = parts[0] + "/" + parts[1]
	return rawState, nil
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
	data.SetId(database + "/" + role)
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
	_, database , err := resourceDatabaseRoleParseId(stateId)

	if err != nil {
		return diag.Errorf("%s",err)
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

	// "name" and "database" are ForceNew, so this always updates the same role identity
	// in place (privileges/inherited roles) rather than dropping and recreating it, which
	// used to revoke the role from every principal holding it, even momentarily.
	err2 := updateRole(client, role, roleList, privileges, database)

	if err2 != nil {
		return diag.Errorf("Could not update the role : %s ", err2)
	}

	// "name" and "database" are ForceNew, so the ID (database/name) cannot have changed -
	// no need to re-set it here.
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
		// The role is gone from MongoDB (e.g. deleted out of band, or the cluster was
		// recreated). Clear the ID instead of erroring so Terraform treats it as absent
		// and offers to recreate it, rather than getting permanently stuck.
		data.SetId("")
		return nil
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
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected database/roleName", id)
	}

	database := parts[0]
	roleName := parts[1]

	return roleName , database , nil
}

