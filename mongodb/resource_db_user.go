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

func resourceDatabaseUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDatabaseUserCreate,
		ReadContext:   resourceDatabaseUserRead,
		UpdateContext: resourceDatabaseUserUpdate,
		DeleteContext: resourceDatabaseUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    resourceDatabaseUserResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceDatabaseUserUpgradeV0,
			},
		},
		Schema: map[string]*schema.Schema{
			"auth_database": {
				Type:     schema.TypeString,
				Required: true,
				// MongoDB's updateUser command cannot move a user to a different auth
				// database - that's a different user identity. Force a create/destroy
				// instead of silently targeting the wrong user on update.
				ForceNew: true,
			},
			"name":{
				Type:     schema.TypeString,
				Required: true,
				// MongoDB has no user-rename command; a name change is a new identity.
				ForceNew: true,
			},
			"password":{
				Type:     schema.TypeString,
				Optional: true,
				// Not required for external-identity auth (X.509 client certs, or IAM auth
				// on AWS DocumentDB via "MONGODB-AWS" in auth_mechanisms) - MongoDB rejects
				// createUser/updateUser calls that include a password for those mechanisms.
			},
			"auth_mechanisms": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				// e.g. ["MONGODB-AWS"] for AWS DocumentDB IAM-authenticated users (name is
				// then the IAM user/role ARN, auth_database "$external", no password). Unset
				// leaves MongoDB's own default (SCRAM) in place.
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



// resourceDatabaseUserResourceV0 is the schema shape prior to the "database/name" ID
// migration (v0 IDs were base64("database.name")). It only needs to describe the schema
// closely enough for StateUpgraders to decode the raw v0 state, so it mirrors the current
// schema rather than being maintained as a historical snapshot.
func resourceDatabaseUserResourceV0() *schema.Resource {
	return &schema.Resource{
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

// resourceDatabaseUserUpgradeV0 rewrites the base64("database.name") ID from schema v0 into
// the plain "database/name" ID used from v1 onward, so the provider upgrade is invisible:
// existing state is migrated in place on the next refresh/apply, with no forced replacement
// and no manual `terraform state` surgery required from the user.
func resourceDatabaseUserUpgradeV0(_ context.Context, rawState map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
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
		return rawState, fmt.Errorf("unexpected format of v0 ID (%s), expected base64(database.username)", oldId)
	}

	rawState["id"] = parts[0] + "/" + parts[1]
	return rawState, nil
}

func resourceDatabaseUserDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client , connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var stateId = data.State().ID
	userName, database, err := resourceDatabaseUserParseId(stateId)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	adminDB := client.Database(database)

	result := adminDB.RunCommand(context.Background(), bson.D{{Key: "dropUser", Value: userName}})
	if result.Err() != nil {
		return diag.Errorf("%s",result.Err())
	}

	return nil
}

func resourceDatabaseUserUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client , connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var stateId = data.State().ID
	_, _, err := resourceDatabaseUserParseId(stateId)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	var userName = data.Get("name").(string)
	var database = data.Get("auth_database").(string)
	var userPassword = data.Get("password").(string)
	var authMechanisms = stringSetToSlice(data.Get("auth_mechanisms").(*schema.Set))

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
	// "name" and "auth_database" are ForceNew, so this always updates the same user
	// identity in place (password/roles) rather than dropping and recreating it, which
	// used to break any connection already authenticated as this user.
	err2 := updateUser(client, user, roleList, authMechanisms, database)
	if err2 != nil {
		return diag.Errorf("Could not update the user : %s ", err2)
	}

	// "name" and "auth_database" are ForceNew, so the ID (database/name) cannot have
	// changed - no need to re-set it here.
	return resourceDatabaseUserRead(ctx, data, i)
}

func resourceDatabaseUserRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client , connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	stateID := data.State().ID
	username, database , err := resourceDatabaseUserParseId(stateID)
	if err != nil {
		return diag.Errorf("%s",err)
	}
	result , decodeError := getUser(client,username,database)
	if decodeError != nil {
		return diag.Errorf("Error decoding user : %s ", err)
	}
	if len(result.Users) == 0 {
		// The user is gone from MongoDB (e.g. deleted out of band, or the cluster was
		// recreated). Clear the ID instead of erroring so Terraform treats it as absent
		// and offers to recreate it, rather than getting permanently stuck.
		data.SetId("")
		return nil
	}
	roles := make([]interface{}, len(result.Users[0].Roles))

	for i, s := range result.Users[0].Roles {
			roles[i] = map[string]interface{}{
				"db": s.Db,
				"role": s.Role,
			}
	}
	dataSetError := data.Set("role", roles)
	if dataSetError != nil  {
		return diag.Errorf("error setting role : %s " , dataSetError)
	}
	dataSetError = data.Set("auth_database", database)
	if dataSetError != nil  {
		return diag.Errorf("error setting auth_db : %s " , dataSetError)
	}
	dataSetError = data.Set("password", data.Get("password"))
	if dataSetError != nil  {
		return diag.Errorf("error setting password : %s " , dataSetError)
	}
	dataSetError = data.Set("auth_mechanisms", result.Users[0].Mechanisms)
	if dataSetError != nil {
		return diag.Errorf("error setting auth_mechanisms : %s ", dataSetError)
	}
	data.SetId(stateID)
	return nil
}

func resourceDatabaseUserCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var config = i.(*MongoDatabaseConfiguration)
	client , connectionError := MongoClientInit(config)
	if connectionError != nil {
		return diag.Errorf("Error connecting to database : %s ", connectionError)
	}
	var database = data.Get("auth_database").(string)
	var userName = data.Get("name").(string)
	var userPassword = data.Get("password").(string)
	var authMechanisms = stringSetToSlice(data.Get("auth_mechanisms").(*schema.Set))
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
	err := createUser(client,user,roleList,authMechanisms,database)
	if err != nil {
		return diag.Errorf("Could not create the user : %s ", err)
	}
	data.SetId(database + "/" + userName)
	return resourceDatabaseUserRead(ctx, data, i)
}

// stringSetToSlice converts a TypeSet-of-strings schema value into a []string, used for
// auth_mechanisms - the mongo driver / bson command needs a concrete []string, not the
// *schema.Set's []interface{} form.
func stringSetToSlice(set *schema.Set) []string {
	raw := set.List()
	out := make([]string, len(raw))
	for i, v := range raw {
		out[i] = v.(string)
	}
	return out
}

func resourceDatabaseUserParseId(id string) (string, string, error){
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected database/username", id)
	}

	database := parts[0]
	userName := parts[1]

	return userName , database , nil
}
